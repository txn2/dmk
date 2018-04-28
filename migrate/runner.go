package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"time"

	"strings"

	"log"

	"github.com/Masterminds/sprig"
	"github.com/boltdb/bolt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mcuadros/go-candyjs"
	"github.com/satori/go.uuid"
	"github.com/txn2/dmk/cfg"
	"github.com/txn2/dmk/driver"
	"github.com/txn2/dmk/tunnel"
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

// Runner runs migrations for a project.
type RunnerCfg struct {
	Project       Project
	DriverManager *driver.Manager
	TunnelManager tunnel.Manager
	DryRun        bool
	Verbose       bool
	Path          string // relative path to config
}

// RunResult is returned by the Run method
type RunResult struct {
	MachineName       string
	SourceArgs        []string
	DestinationDriver *driver.Driver
	SourceDriver      *driver.Driver
	Done              chan bool
	Error             chan error
	Status            chan string
}

// runner runs migrations for a project with the Run method.
type runner struct {
	cfg      RunnerCfg
	drivers  map[string]driver.Driver // store configured drivers
	localDbs map[string]*bolt.DB      // local bold databases for value mapping
}

// NewRunner configures and returns a new runner
func NewRunner(cfg RunnerCfg) *runner {

	runner := &runner{
		cfg: cfg,
	}

	return runner
}

// getLocalDb gets the database
func (r *runner) getLocalDb(migration string) (*bolt.DB, error) {
	// one database per migration (to avoid dealing with multiple writers)
	dbFile := r.cfg.Path + r.cfg.Project.Component.MachineName + "-" + migration + ".db"

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
func (r *runner) configureDriver(migration string, db cfg.Database) (driver.Driver, error) {
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
	d, err := r.cfg.DriverManager.GetNewDriver(db.Driver)
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
func (r *runner) tunnel(database cfg.Database) error {
	// setup a tunnel if needed
	if database.Tunnel != "" {
		if tunnelCfg, ok := r.cfg.Project.Tunnels[database.Tunnel]; ok {
			err := r.cfg.TunnelManager.Tunnel(tunnelCfg)
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

// Run runs a migration
func (r *runner) RunAsync(machineName string, sourceArgs []string) (*RunResult, error) {
	doneChan := make(chan bool, 1)
	errorChan := make(chan error, 1)
	statusChan := make(chan string, 1)

	//	doneChan <- false

	runResult := &RunResult{
		MachineName: machineName,
		SourceArgs:  sourceArgs,
		Done:        doneChan,
		Error:       errorChan,
		Status:      statusChan,
	}

	go r.run(runResult)

	return runResult, nil
}

// run a migration (see RunAsync)
func (r *runner) run(runResult *RunResult) {
	machineName := runResult.MachineName
	sourceArgs := runResult.SourceArgs

	runResult.Status <- fmt.Sprintln("Running Migration: " + machineName)

	if r.cfg.DryRun {
		runResult.Status <- fmt.Sprintf("\n>> This is a DRY RUN. No data will be migrated. <<\n\n")
	}

	// get the migration
	migration, ok := r.cfg.Project.Migrations[machineName]
	if ok != true {
		runResult.Error <- errors.New("no migration found for " + machineName)
		return
	}

	// get the source db
	sourceDb, ok := r.cfg.Project.Databases[migration.SourceDb]
	if ok != true {
		runResult.Error <- errors.New("no source database found for " + migration.SourceDb)
		return
	}

	err := r.tunnel(sourceDb)
	if err != nil {
		runResult.Error <- errors.New("unable to tunnel for " + sourceDb.Component.Name)
		return
	}

	// get a driver for the source of migration
	sourceDriver, err := r.configureDriver(machineName, sourceDb)
	if err != nil {
		runResult.Error <- err
		return
	}

	// set a pointer to the source driver in the run result
	runResult.SourceDriver = &sourceDriver

	runResult.Status <- fmt.Sprintf("%s source query expects %d args.\n", machineName, migration.SourceQueryNArgs)
	runResult.Status <- fmt.Sprintf("%s received %d args.\n", machineName, len(sourceArgs))

	if migration.SourceQueryNArgs != len(sourceArgs) {
		runResult.Error <- fmt.Errorf("expecting %d args and got %d", migration.SourceQueryNArgs, len(sourceArgs))
		return
	}

	// Source data collection.
	// do we have the requested number of args
	runResult.Status <- fmt.Sprintf("Migration %s Source Query: %s\n", machineName, strings.Trim(migration.SourceQuery, "\n"))
	runResult.Status <- fmt.Sprintf("Migration %s Source Args: %s\n", machineName, sourceArgs)

	sourceRecordChan, err := sourceDriver.Out(migration.SourceQuery, sourceArgs)
	if err != nil {
		runResult.Error <- err
		return
	}

	runResult.Status <- fmt.Sprintf("Migration DestinationDb: %s\n", migration.DestinationDb)

	destinationDb, ok := r.cfg.Project.Databases[migration.DestinationDb]
	if ok != true {
		runResult.Error <- err
		return
	}

	// get the destination driver
	runResult.Status <- fmt.Sprintf("Migration Driver: %s\n", destinationDb.Driver)

	destinationDriver, err := r.configureDriver(machineName, destinationDb)
	if err != nil {
		runResult.Error <- err
		return
	}
	runResult.DestinationDriver = &destinationDriver

	// javascript transformation script
	script := migration.TransformationScript

	var ctx *candyjs.Context

	if script != "" {
		// Javascript engine,
		// see http://duktape.org/ and https://github.com/olebedev/go-duktape
		// see https://github.com/mcuadros/go-candyjs
		ctx = candyjs.NewContext()

		defer ctx.DestroyHeap()
		r.addScriptFunctions(*ctx, machineName)
	}

	queryTemplate, err := template.New("query").Funcs(sprig.TxtFuncMap()).Parse(migration.DestinationQuery)
	if err != nil {
		// we have no intention from recovering from this.
		// todo: determine why we failed.
		panic(err)
	}

	runResult.Status <- fmt.Sprintf("Migrating data from %s to %s.\n", migration.SourceDb, migration.DestinationDb)

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

			// Evaluate the javascript script
			ctx.EvalString(script)

			// If the transformation script wants us to skip this record
			if skipRecord {
				runResult.Status <- fmt.Sprintf("Migration script wants to skip this record.\n")
				continue
			}

			// If the transformation script wants us to the the migration
			if endMigration {
				runResult.Status <- fmt.Sprintf("Migration script is terminating migration.\n")
				runResult.Done <- true
				return
			}
		}

		if r.cfg.DryRun {
			continue
		}

		var query bytes.Buffer
		err = queryTemplate.Execute(&query, record)
		if err != nil {
			runResult.Error <- err
			runResult.Done <- true
			return
		}

		runResult.Status <- fmt.Sprintf("Migration %s Destination Args: %s\n", machineName, args)

		err = destinationDriver.In(query.String(), args, record)
		if err != nil {
			runResult.Error <- err
			runResult.Done <- true
			return
		}
	}

	destinationDriver.Done()
	runResult.Status <- fmt.Sprintf("Done with migration %s\n", migration.Component.MachineName)
	runResult.Done <- true
}

// addScriptFunctions add utility functions to script context
func (r *runner) addScriptFunctions(ctx candyjs.Context, machineName string) {

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
		log.Printf("<script dump> %s: %s", machineName, sd)
	})

	ctx.PushGlobalGoFunction("print", func(obj interface{}) {
		log.Printf("<script message> %s: %s", machineName, obj)
	})
}

// persistVal gets or stores a fallback value
func (r *runner) persistVal(migration string, k string, fallback string) string {
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
func (r *runner) scriptRunner(machineNameFromScript string, argsFromScript []string, cb func(string)) []driver.ResultCollectionItem {
	// the callback is optional for javascript
	if cb == nil {
		cb = func(string) {
			// suppress
		}
	}

	runResult, err := r.RunAsync(machineNameFromScript, argsFromScript)
	if err != nil {
		cb(fmt.Sprintf("ERROR: %s\n", err.Error()))
	}

	// script run migrations are synchronous so we need to
	// wait for done
	cb(fmt.Sprintf("Run migration called from script. Running...\n"))

migrationOut:
	for {
		select {
		case <-runResult.Done:
			cb(fmt.Sprintf("[%s]: Done", machineNameFromScript))
			break migrationOut
		case msg := <-runResult.Status:
			cb(fmt.Sprintf("[%s]: %s", machineNameFromScript, msg))
		case err := <-runResult.Error:
			cb(fmt.Sprintf("ERROR [%s]: %s", machineNameFromScript, err.Error()))
		}
	}

	dd := *runResult.DestinationDriver

	if cdd, ok := dd.(*driver.Collector); ok {
		collection := cdd.GetCollection()
		if r.cfg.Verbose {
			cb(fmt.Sprintf("Argset will receive %d items from collector.", len(collection)))
		}

		return collection
	}

	cb(fmt.Sprintf("WARNING: run() for %s executed from a script but did not output to a collector.", machineNameFromScript))

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
