package cmd

import (
	"bytes"
	"fmt"
	"html/template"

	"errors"

	"github.com/cjimti/migration-kit/driver"
	"github.com/desertbit/grumble"
	"github.com/robertkrimen/otto"
)

func init() {
	runCmd := &grumble.Command{
		Name:      "run",
		Help:      "run a migration",
		Usage:     "run [MIGRATION]",
		Aliases:   []string{"r"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {

				if len(c.Args) == 1 {
					runMigration(c.Args[0])
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

func runMigration(machineName string) {

	fmt.Println("Running Migration: " + machineName)

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

	// get a driver for the type and configure it
	sourceDriver, err := DriverManager.GetNewDriver(sourceDb.Driver)
	if err != nil {
		App.PrintError(err)
		return
	}

	sourceDriver.Configure(sourceDb.Configuration)
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

	// javascript virtual machine
	script := migration.TransformationScript
	jsVm := otto.New()

	fmt.Printf("Migrating data from %s to %s.\n", migration.SourceDb, migration.DestinationDb)
	for r := range sourceRecordChan {

		tmpl, err := template.New("test").Parse(migration.DestinationQuery)
		if err != nil {
			panic(err)
		}

		var query bytes.Buffer

		if script != "" {
			jsVm.Set("data", r)
			jsVm.Run(script)

			if value, err := jsVm.Get("data"); err == nil {
				retRec, _ := value.Export()
				r = retRec.(driver.Record)
			}

		}

		err = tmpl.Execute(&query, r)

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
