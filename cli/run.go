package cli

import (
	"fmt"
	"log"

	"io/ioutil"

	"time"

	"github.com/desertbit/grumble"
	"github.com/gizak/termui"
	"github.com/txn2/dmk/migrate"
	"github.com/txn2/dmk/panel"
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
			f.Bool("l", "logout", false, "Log style output, no GUI.")
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

// runMigration runs a migration
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

	res, err := runner.RunAsync(machineName, args)
	if err != nil {
		Cli.PrintError(err)
	}

	if f.Bool("logout") {
		runMigrationLogOut(res)
		return
	}

	runMigrationGui(res)
}

// runMigrationLogOut plain log output if requested by
// the -l flag.
func runMigrationLogOut(res *migrate.RunResult) {
	for {
		select {
		case <-res.Done:
			log.Printf("Done.")
			return
		case msg := <-res.Status: // TODO log output?
			log.Printf(msg.Msg)
		case err := <-res.Error:
			fmt.Printf("Got error: %s", err.Error())
			return

		}
	}
}

// runMigrationGui uses termui to display migration status
func runMigrationGui(res *migrate.RunResult) {
	err := termui.Init()

	if err != nil {
		panic(err)
	}
	defer termui.Close()

	// handle key q pressing
	termui.Handle("/sys/kbd/q", func(termui.Event) {
		// press q to quit
		termui.StopLoop()
	})

	prog := termui.NewPar("")
	prog.Height = 3
	prog.BorderLabel = "Completed"

	per := termui.NewPar("")
	per.Height = 3
	per.BorderLabel = "Per Record"

	tot := termui.NewPar("")
	tot.Height = 3
	tot.BorderLabel = "Elapsed"

	ins := termui.NewPar("Press [q] to quit at any time.")
	ins.Height = 1
	ins.Border = false

	status := panel.NewLogPanel("Migration Status", 8)

	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(12, 0, status.Panel),
		),
		termui.NewRow(
			termui.NewCol(2, 0, prog),
			termui.NewCol(2, 0, per),
			termui.NewCol(2, 0, tot),
		),
		termui.NewRow(
			termui.NewCol(12, 0, ins),
		),
	)

	termui.Body.Align()

	termui.Render(ins, prog, per, tot)

	// use the -l (logout) flag if script output is needed.
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	start := time.Now()

	// @todo: fix race condition from termui when using this as a goroutine
	// trade off is not using a routine here not being able to use termui.Loop to
	// wait for the "q" command
	go func() {
		for {
			select {
			case <-res.Done:
				status.AddMessage(fmt.Sprintf("Done."))
				return
			case msg := <-res.Status:
				elapsed := time.Since(start)
				perR := elapsed.Seconds() / float64(msg.Count)

				status.AddMessage(msg.Msg)

				prog.Text = fmt.Sprintf("%d", msg.Count)
				tot.Text = fmt.Sprintf("%.2fs", elapsed.Seconds())
				per.Text = fmt.Sprintf("%.2fs", perR)

				termui.Render(status.Panel, prog, per, tot)
			case err := <-res.Error:
				status.AddMessage(fmt.Sprintf("Got error: %s", err.Error()))
				return
			}
			termui.Render(status.Panel)
		}
	}()

	termui.Loop()
}
