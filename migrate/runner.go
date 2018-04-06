package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"time"

	"github.com/Masterminds/sprig"
	"github.com/cjimti/migration-kit/cfg"
	"github.com/cjimti/migration-kit/driver"
	"github.com/cjimti/migration-kit/tunnel"
	"github.com/davecgh/go-spew/spew"
	"github.com/mcuadros/go-candyjs"
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

type Runner struct {
	Project       Project
	DriverManager *driver.Manager
	TunnelManager tunnel.Manager
	DryRun        bool
	Verbose       bool
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

	// get a driver for the type and configure it
	sourceDriver, err := r.DriverManager.GetNewDriver(sourceDb.Driver)
	if err != nil {
		return runResult, err
	}
	runResult.SourceDriver = &sourceDriver

	err = sourceDriver.Configure(sourceDb.Configuration)
	if err != nil {
		return runResult, err
	}

	fmt.Printf("Source expects %d args.\n", migration.SourceQueryNArgs)
	fmt.Printf("Received %d args.\n", len(sourceArgs))

	if migration.SourceQueryNArgs != len(sourceArgs) {
		return runResult, errors.New(fmt.Sprintf("expecting %d args and got %d", migration.SourceQueryNArgs, len(sourceArgs)))
	}

	// Source data collection.
	//
	//
	// do we have the requested number of args
	sourceRecordChan, err := sourceDriver.Out(migration.SourceQuery, sourceArgs)
	if err != nil {
		return runResult, err
	}

	// get the destination db
	destinationDb, ok := r.Project.Databases[migration.DestinationDb]
	if ok != true {
		return runResult, errors.New("no destination database found for " + migration.DestinationDb)
	}

	// get the destination driver
	destinationDriver, err := r.DriverManager.GetNewDriver(destinationDb.Driver)
	if err != nil {
		return runResult, err
	}
	runResult.DestinationDriver = &destinationDriver

	// configure destination driver
	destinationDriver.Configure(destinationDb.Configuration)

	// javascript transformation script
	script := migration.TransformationScript

	// Javascript engine,
	// see http://duktape.org/ and https://github.com/olebedev/go-duktape
	// see https://github.com/mcuadros/go-candyjs
	ctx := candyjs.NewContext()
	defer ctx.DestroyHeap()
	storage := make(map[string]interface{})

	queryTemplate, err := template.New("query").Funcs(sprig.TxtFuncMap()).Parse(migration.DestinationQuery)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Migrating data from %s to %s.\n", migration.SourceDb, migration.DestinationDb)

	// iterate over the sourceRecordChan for driver.Record objects
	for record := range sourceRecordChan {

		// todo: ensure the correct number of args
		args := make([]string, 0)

		// modify r, driver.Record
		if script != "" {
			skipRecord := false
			endMigration := false

			ctx.PushGlobalGoFunction("getStorage", func() *map[string]interface{} {
				return &storage
			})

			ctx.PushGlobalGoFunction("sendStorage", func(s map[string]interface{}) {
				storage = s
			})

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

			ctx.PushGlobalGoFunction("dump", func(obj interface{}) {
				spew.Dump(obj)
			})

			// recursive migration (sub query) mainly for used with
			// migrations that migrate to a collector
			ctx.PushGlobalGoFunction("run", func(machineNameFromScript string, argsFromScript []string) []driver.ResultCollectionItem {
				runner := &Runner{
					Project:       r.Project,
					DriverManager: r.DriverManager,
					TunnelManager: r.TunnelManager,
					DryRun:        r.DryRun,
					Verbose:       r.Verbose,
				}

				runResult, err := runner.Run(machineNameFromScript, argsFromScript)
				if err != nil {
					fmt.Printf("ERROR: %s\n", err.Error())
				}

				dd := *runResult.DestinationDriver

				if cdd, ok := dd.(*driver.Collector); ok {
					fmt.Printf("We are a collector!\n")
					collection := cdd.GetCollection()
					spew.Dump(collection)
					return collection
				}

				return []driver.ResultCollectionItem{}
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
			fmt.Printf("Out Query: %s\n", query.String())
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
