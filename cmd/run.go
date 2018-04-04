package cmd

import (
	"fmt"

	"github.com/cjimti/migration-kit/migrate"
	"github.com/desertbit/grumble"
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

	runner := &migrate.Runner{
		Project:       global.Project,
		MachineName:   machineName,
		DriverManager: DriverManager,
		TunnelManager: TunnelManager,
		DryRun:        f.Bool("dry-run"),
	}

	err := runner.Run()
	if err != nil {
		App.PrintError(err)
	}

}
