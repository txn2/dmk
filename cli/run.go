package cli

import (
	"fmt"
	"os"

	"github.com/desertbit/grumble"
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
			f.Bool("l", "log-out", true, "No log file. Log standard out.")
			f.Bool("q", "quiet", false, "No file logging. Sample status.")
			f.Int("", "limit", 0, "Limit the number of records to process.")
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

// runMigration
func runMigration(machineName string, f grumble.FlagMap, args []string) {

	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()

	if f.Bool("no-time") {
		encoderCfg.TimeKey = "" // disable timestamps for deterministic output.
	}

	var out zapcore.WriteSyncer
	out = zapcore.Lock(os.Stdout)

	if f.Bool("log-out") != true && f.Bool("quiet") != true {
		// log gui data to a file
		fl, err := fileLog(machineName)
		if err != nil {
			panic(err)
		}
		defer fl.Close()
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
		Quiet:         f.Bool("quiet"),
		Limit:         f.Int("limit"),
		Logger:        logger,
	}

	rnr := migrate.NewRunner(runnerCfg)
	_, err := rnr.Run(machineName, args)
	if err != nil {
		Cli.PrintError(err)
	}

}
