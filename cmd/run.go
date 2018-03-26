package cmd

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"errors"

	"github.com/Masterminds/sprig"
	"github.com/cjimti/migration-kit/driver"
	"github.com/desertbit/grumble"
	"github.com/mcuadros/go-candyjs"
)

func init() {
	runCmd := &grumble.Command{
		Name:      "run",
		Help:      "run a migration",
		Usage:     "run [MIGRATION]",
		Aliases:   []string{"r"},
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.Bool("d", "dry-run", false, "Dry run outputs the first 5 records.")
		},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {

				if len(c.Args) == 1 {
					runMigration(c.Args[0], c.Flags)
					return nil
				}
				fmt.Printf("Try: %s\n", c.Command.Usage)
				fmt.Printf("Try: \"ls m\" for a list or migrations.\n")
				return nil

			}
			return nil
		},
	}

	App.AddCommand(runCmd)

}

func runMigration(machineName string, f grumble.FlagMap) {

	dryRun := f.Bool("dry-run")

	fmt.Println("Running Migration: " + machineName)

	if dryRun {
		fmt.Printf("\n>> This is a DRY RUN. No data will be migrated. <<\n\n")
	}

	// get the migration
	migration, ok := global.Project.Migrations[machineName]
	if ok != true {
		App.PrintError(errors.New("no migration found for " + machineName))
		return
	}

	// get the source db
	sourceDb, ok := global.Project.Databases[migration.SourceDb]
	if ok != true {
		App.PrintError(errors.New("no source database found for " + migration.SourceDb))
		return
	}

	// setup a tunnel if needed
	if sourceDb.Tunnel != "" {
		if tunnelCfg, ok := global.Project.Tunnels[sourceDb.Tunnel]; ok {
			err := TunnelManager.Tunnel(tunnelCfg)
			if err != nil {
				App.PrintError(err)
				return
			}

			// take a breath while our tunnel connects
			// TODO: detect connection
			time.Sleep(2 * time.Second)
		}

	}

	// get a driver for the type and configure it
	sourceDriver, err := DriverManager.GetNewDriver(sourceDb.Driver)
	if err != nil {
		App.PrintError(err)
		return
	}

	err = sourceDriver.Configure(sourceDb.Configuration)
	if err != nil {
		App.PrintError(err)
		return
	}

	sourceRecordChan, err := sourceDriver.Out(migration.SourceQuery, driver.Args{})
	if err != nil {
		App.PrintError(err)
		return
	}

	// get the destination db
	destinationDb, ok := global.Project.Databases[migration.DestinationDb]
	if ok != true {
		App.PrintError(errors.New("no destination database found for " + migration.DestinationDb))
		return
	}

	// get the destination driver
	destinationDriver, err := DriverManager.GetNewDriver(destinationDb.Driver)
	if err != nil {
		App.PrintError(err)
		return
	}

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

	queryTemplate, err := template.New("query").Funcs(sprig.FuncMap()).Parse(migration.DestinationQuery)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Migrating data from %s to %s.\n", migration.SourceDb, migration.DestinationDb)

	// iterate over the sourceRecordChan for driver.Record objects
	for record := range sourceRecordChan {

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

			ctx.PushGlobalGoFunction("skip", func() {
				skipRecord = true
			})

			ctx.PushGlobalGoFunction("end", func() {
				endMigration = true
			})

			ctx.EvalString(migration.TransformationScript)

			// If the transformation script wants us to skip this record
			if skipRecord {
				continue
			}

			// If the transformation script wants us to skip this record
			if endMigration {
				return
			}
		}

		if dryRun {
			return
		}

		var query bytes.Buffer
		err = queryTemplate.Execute(&query, record)
		if err != nil {
			App.PrintError(err)
			return
		}

		err = destinationDriver.In(query.String())
		if err != nil {
			App.PrintError(err)
			return
		}
	}

	destinationDriver.Done()

	fmt.Printf("Done with migration %s\n", migration.Component.MachineName)

}
