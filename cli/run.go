package cli

import (
	"fmt"

	"log"

	"github.com/desertbit/grumble"
	"github.com/txn2/dmk/migrate"
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
			f.Bool("v", "verbose", false, "Verbose output.")
		},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {

				if len(c.Args) > 0 {
					runMigration(c.Args[0], c.Flags, c.Args[1:])
					return nil
				}
				fmt.Printf("Try: %s\n", c.Command.Usage)
				fmt.Printf("Try: \"ls m\" for a list or migrations.\n")
				return nil

			}
			return nil
		},
	}

	Cli.AddCommand(runCmd)

}

func runMigration(machineName string, f grumble.FlagMap, args []string) {

	runnerCfg := migrate.RunnerCfg{
		Project:       appState.Project,
		DriverManager: DriverManager,
		TunnelManager: TunnelManager,
		Path:          appState.Directory,
		DryRun:        f.Bool("dry-run"),
		Verbose:       f.Bool("verbose"),
	}

	runner := migrate.NewRunner(runnerCfg)

	// todo: display stats when the result becomes useful
	res, err := runner.RunAsync(machineName, args)
	if err != nil {
		Cli.PrintError(err)
	}

	for {
		select {
		case <-res.Done:
			return
		case msg := <-res.Status: // TODO log output?
			log.Printf("%s", msg)
		case err := <-res.Error:
			fmt.Printf("Got error: %s", err.Error())
		}
	}

}
