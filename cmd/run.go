package cmd

import (
	"fmt"

	"errors"

	"github.com/davecgh/go-spew/spew"
	"github.com/desertbit/grumble"
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
		App.PrintError(errors.New("no database found for " + machineName))
		return
	}

	// get a driver for the type and configure it
	sourceDriver, err := DriverManager.GetNewDriver(sourceDb.Driver)
	if err != nil {
		App.PrintError(err)
		return
	}

	sourceDriver.Configure(sourceDb.Configuration)
	//sourceDriver.Execute()

	spew.Dump(sourceDriver)

	// ask the db for a configured Driver
	//fmt.Printf("Source Driver: %s\n", sourceDb.Driver)
	//fmt.Printf("Source Config: %s\n", sourceDb.Configuration)

	// execute the source query

}
