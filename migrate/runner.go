package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"time"

	"strings"

	"github.com/Masterminds/sprig"
	"github.com/boltdb/bolt"
	"github.com/cjimti/dmk/cfg"
	"github.com/cjimti/dmk/driver"
	"github.com/cjimti/dmk/tunnel"
	"github.com/davecgh/go-spew/spew"
	"github.com/mcuadros/go-candyjs"
	"github.com/satori/go.uuid"
)

// A Runner runs a Migration consisting of a
//  - source DB,
//  - source query,
//  - destination DB
//  - destination query.
//  - a collection of transformers
//
// 1) A query is executed on the on source db and foreach result a transformers is invoked along with the
// collection of transformers.
//
// 2) Each source result becomes transformed data (map[string]interface) and is then used as arguments along with
// a query against the destination database.

// Runner runs migrations.
type Runner struct {
	Project       Project
	DriverManager *driver.Manager
	TunnelManager tunnel.Manager
	DryRun        bool
	Verbose       bool
	Path          string                   // relative path to config
	drivers       map[string]driver.Driver // store configured drivers
	localDbs      map[string]*bolt.DB      // local bold databases for value mapping
}

// getLocalDb gets the database
func (r *Runner) getLocalDb(migration string) (*bolt.DB, error) {
	// one database per migration (to avoid dealing with multiple writers)
	dbFile := r.Path + r.Project.Component.MachineName + "-" + migration + ".db"

	if db, ok := r.localDbs[dbFile]; ok {
		return db, nil
	}

	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	if r.localDbs == nil {
		r.localDbs = make(map[string]*bolt.DB, 1)
	}
	r.localDbs[dbFile] = db

	return db, nil
}

// configureDriver configures a driver for the migration and database. The configured
// drivers is stored in the event it needs to be re-used in a sub migration.
func (r *Runner) configureDriver(migration string, db cfg.Database) (driver.Driver, error) {
	key := migration + "_" + db.Component.MachineName

	if r.drivers == nil {
		r.drivers = make(map[string]driver.Driver, 0)
	}

	// do we have a configured driver for this migration and database?
	if d, ok := r.drivers[key]; ok {
		d.Init()
		return d, nil
	}

	// get a driver of the specified type
	d, err := r.DriverManager.GetNewDriver(db.Driver)
	if err != nil {
		return nil, err
	}

	// configure the driver
	err = d.Configure(db.Configuration)
	if err != nil {
		return nil, err
	}

	// store the driver
	r.drivers[key] = d

	d.Init()
	return d, nil
}

// tunnel if needed
func (r *Runner) tunnel(database cfg.Database) error {
	// setup a tunnel if needed
	if database.Tunnel != "" {
		if tunnelCfg, ok := r.Project.Tunnels[database.Tunnel]; ok {
			err := r.TunnelManager.Tunnel(tunnelCfg)
			if err != nil {
				return err
			}

			// take a breath while our tunnel connects
			// TODO: detect connection
			time.Sleep(2 * time.Second)
		}

	}

	return nil
}

// RunResult is returned by the Run method
type RunResult struct {
	MachineName       string
	SourceArgs        []string
	DestinationDriver *driver.Driver
	SourceDriver      *driver.Driver
}

// Run runs a migration
func (r *Runner) Run(machineName string, sourceArgs []string) (*RunResult, error) {
	runResult := &RunResult{
		MachineName: machineName,
		SourceArgs:  sourceArgs,
	}

	fmt.Println("Running Migration: " + machineName)

	if r.DryRun {
		fmt.Printf("\n>> This is a DRY RUN. No data will be migrated. <<\n\n")
	}

	// get the migration
	migration, ok := r.Project.Migrations[machineName]
	if ok != true {
		return runResult, errors.New("no migration found for " + machineName)
	}

	// get the source db
	sourceDb, ok := r.Project.Databases[migration.SourceDb]
	if ok != true {
		return runResult, errors.New("no source database found for " + migration.SourceDb)
	}

	err := r.tunnel(sourceDb)
	if err != nil {
		return runResult, errors.New("unable to tunnel for " + sourceDb.Component.Name)
	}

	// get a driver for the source of migration
	sourceDriver, err := r.configureDriver(machineName, sourceDb)
	if err != nil {
		return runResult, err
	}

	// set a pointer to the source driver in the run result
	runResult.SourceDriver = &sourceDriver

	if r.Verbose {
		fmt.Printf("%s source query expects %d args.\n", machineName, migration.SourceQueryNArgs)
		fmt.Printf("%s received %d args.\n", machineName, len(sourceArgs))
	}

	if migration.SourceQueryNArgs != len(sourceArgs) {
		return runResult, fmt.Errorf("expecting %d args and got %d", migration.SourceQueryNArgs, len(sourceArgs))
	}

	// Source data collection.
	// do we have the requested number of args
	if r.Verbose {
		fmt.Printf("Migration %s Source Query: %s\n", machineName, strings.Trim(migration.SourceQuery, "\n"))
		fmt.Printf("Migration %s Source Args: %s\n", machineName, sourceArgs)
	}
	sourceRecordChan, err := sourceDriver.Out(migration.SourceQuery, sourceArgs)
	if err != nil {
		return runResult, err
	}

	// get the destination db
	if r.Verbose {
		fmt.Printf("Migration DestinationDb: %s\n", migration.DestinationDb)
	}
	destinationDb, ok := r.Project.Databases[migration.DestinationDb]
	if ok != true {
		return runResult, errors.New("no destination database found for " + migration.DestinationDb)
	}

	// get the destination driver
	if r.Verbose {
		fmt.Printf("Migration Driver: %s\n", destinationDb.Driver)
	}

	destinationDriver, err := r.configureDriver(machineName, destinationDb)
	if err != nil {
		return runResult, err
	}
	runResult.DestinationDriver = &destinationDriver

	// javascript transformation script
	script := migration.TransformationScript

	// Javascript engine,
	// see http://duktape.org/ and https://github.com/olebedev/go-duktape
	// see https://github.com/mcuadros/go-candyjs
	ctx := candyjs.NewContext()
	defer ctx.DestroyHeap()
	r.addScriptFunctions(*ctx, machineName)

	queryTemplate, err := template.New("query").Funcs(sprig.TxtFuncMap()).Parse(migration.DestinationQuery)
	if err != nil {
		panic(err)
	}

	if r.Verbose {
		fmt.Printf("Migrating data from %s to %s.\n", migration.SourceDb, migration.DestinationDb)
	}

	// iterate over the sourceRecordChan for driver.Record objects
	for record := range sourceRecordChan {

		args := make([]string, 0)

		// modify r, driver.Record
		if script != "" {
			skipRecord := false
			endMigration := false

			ctx.PushGlobalGoFunction("getRecord", func() driver.Record {
				return record
			})

			ctx.PushGlobalGoFunction("sendRecord", func(r driver.Record) {
				record = r
			})

			ctx.PushGlobalGoFunction("sendArgs", func(a []string) {
				args = a
			})

			ctx.PushGlobalGoFunction("skip", func() {
				skipRecord = true
			})

			ctx.PushGlobalGoFunction("end", func() {
				endMigration = true
			})

			ctx.EvalString(script)

			// If the transformation script wants us to skip this record
			if skipRecord {
				continue
			}

			// If the transformation script wants us to skip this record
			if endMigration {
				return runResult, nil
			}
		}

		if r.DryRun {
			continue
		}

		var query bytes.Buffer
		err = queryTemplate.Execute(&query, record)
		if err != nil {
			return runResult, err
		}

		if r.Verbose {
			fmt.Printf("Migration %s Destination Query: %s\n", machineName, strings.Trim(query.String(), "\n"))
			fmt.Printf("Migration %s Destination Args: %s\n", machineName, args)
		}

		err = destinationDriver.In(query.String(), args, record)
		if err != nil {
			return runResult, err
		}
	}

	destinationDriver.Done()

	fmt.Printf("Done with migration %s\n", migration.Component.MachineName)

	return runResult, nil
}

// addScriptFunctions add utility functions to script context
func (r *Runner) addScriptFunctions(ctx candyjs.Context, machineName string) {

	// memory storage
	storage := make(map[string]interface{})

	// recursive migration (sub query) mainly for used with
	// migrations that migrate to a collector
	ctx.PushGlobalGoFunction("run", r.scriptRunner)

	// persistent storage for value maps
	ctx.PushGlobalGoFunction("persistVal", r.persistVal)

	ctx.PushGlobalGoFunction("getMigration", func() string {
		return machineName
	})

	ctx.PushGlobalGoFunction("getUuid", func() string {
		return uuid.NewV4().String()
	})

	ctx.PushGlobalGoFunction("getStorage", func() *map[string]interface{} {
		return &storage
	})

	ctx.PushGlobalGoFunction("sendStorage", func(s map[string]interface{}) {
		storage = s
	})

	ctx.PushGlobalGoFunction("dump", func(obj interface{}) {
		sd := spew.Sdump(obj)
		fmt.Printf(">>> SCRIPT %s: %s\n", machineName, sd)
	})

	ctx.PushGlobalGoFunction("print", func(obj interface{}) {
		fmt.Printf(">>> SCRIPT %s: %s\n", machineName, obj)
	})
}

// persistVal gets or stores a fallback value
func (r *Runner) persistVal(migration string, k string, fallback string) string {
	db, err := r.getLocalDb(migration)
	if err != nil {
		fmt.Printf("LOCAL DB ERROR: %s", err.Error())
		return ""
	}
	var retVal []byte

	bucket := "persistVal"

	found := false

	// ensure that we have a bucket
	ensureBucket(db, bucket)

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v := b.Get([]byte(k))

		if v != nil {
			retVal = make([]byte, len(v))
			copy(retVal, v)
			found = true
			return nil
		}

		return nil
	})

	if found == true {
		return string(retVal)
	}

	go func() {
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucket))
			err := b.Put([]byte(k), []byte(fallback))
			return err
		})
	}()

	return fallback
}

// scriptRunner returns run function for script context
func (r *Runner) scriptRunner(machineNameFromScript string, argsFromScript []string) []driver.ResultCollectionItem {
	runResult, err := r.Run(machineNameFromScript, argsFromScript)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	dd := *runResult.DestinationDriver

	if cdd, ok := dd.(*driver.Collector); ok {
		collection := cdd.GetCollection()
		if r.Verbose {
			fmt.Printf("Argset will receive %d items from collector.\n", len(collection))
		}

		return collection
	}

	if r.Verbose {
		fmt.Printf("WARNING: run() for %s executed from a script but did not output to a collector.\n", machineNameFromScript)
	}
	return []driver.ResultCollectionItem{}
}

// ensureBucket makes a bucket if one does not exist
func ensureBucket(db *bolt.DB, bucket string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return err
}
