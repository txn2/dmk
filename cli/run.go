package cli

import (
	"fmt"
	"os"

	"log"

	"sync"

	"github.com/desertbit/grumble"
	"github.com/jroimartin/gocui"
	"github.com/txn2/dmk/migrate"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
			f.Bool("n", "no-time", false, "Disable timestamps and duration for deterministic output.")
			f.Bool("l", "log-out", false, "No GUI and no log file. Log standard out.")
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

// GuiDataCfg
type GuiDataCfg struct {
	machineName string
	fl          *os.File
}

// guiData
type guiData struct {
	cfg *GuiDataCfg
}

// NewGui creates a CLI GUI for migration data
func NewGui(cfg *GuiDataCfg) (*guiData, *sync.WaitGroup) {
	gui := &guiData{
		cfg: cfg,
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(group *sync.WaitGroup) {
		g, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			log.Panicln(err)
		}
		defer g.Close()

		g.SetManagerFunc(guiLayout)

		if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, guiQuit); err != nil {
			log.Panicln(err)
		}

		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Panicln(err)
		}

		group.Done()
	}(&wg)

	return gui, &wg
}

// guiLayout
func guiLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("hello", maxX/2-7, maxY/2, maxX/2+7, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "Hello world!")
	}
	return nil
}

// guiQuit
func guiQuit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// Write implements SyncWriter
func (g *guiData) Write(p []byte) (nn int, err error) {

	if g.cfg.fl != nil {
		g.cfg.fl.Write(p)
	}

	//log.Printf("Line: %s", p)
	return
}

// Sync implements SyncWriter
func (g *guiData) Sync() error {
	g.cfg.fl.Sync()
	return nil
}

// runMigration
func runMigration(machineName string, f grumble.FlagMap, args []string) {

	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()

	if f.Bool("no-time") {
		encoderCfg.TimeKey = "" // disable timestamps for deterministic output.
	}

	// log gui data to a file
	fl, err := fileLog(machineName)
	if err != nil {
		panic(err)
	}
	defer fl.Close()

	gui, wg := NewGui(&GuiDataCfg{machineName, fl})
	out := zapcore.Lock(gui)

	if f.Bool("log-out") {
		out = zapcore.Lock(os.Stdout)
	}

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		out,
		atom,
	))

	atom.SetLevel(zap.DebugLevel)
	defer logger.Sync()

	runnerCfg := migrate.RunnerCfg{
		Project:       appState.Project,
		DriverManager: DriverManager,
		TunnelManager: TunnelManager,
		Path:          appState.Directory,
		NoTime:        f.Bool("no-time"),
		DryRun:        f.Bool("dry-run"),
		Verbose:       f.Bool("verbose"),
		Logger:        logger,
	}

	rnr := migrate.NewRunner(runnerCfg)

	// todo: display stats when the result becomes useful
	_, err = rnr.Run(machineName, args)
	if err != nil {
		Cli.PrintError(err)
	}

	wg.Wait()
}
