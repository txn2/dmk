package cmd

import (
	"fmt"

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
				return nil

			}
			return nil
		},
	}

	App.AddCommand(runCmd)

}

func runMigration(machineName string) {

	fmt.Println("Running Migration: " + machineName)
}
