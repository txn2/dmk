package cmd

import (
	"fmt"

	"github.com/desertbit/grumble"
	"github.com/fatih/color"
)

var global struct {
	Env map[string]string
}

var App = grumble.New(&grumble.Config{
	Name:                  "DMK",
	Description:           "Data migration kit.",
	HistoryFile:           "/tmp/dmk.hist",
	Prompt:                "dmk » ",
	PromptColor:           color.New(color.FgGreen, color.Bold),
	HelpHeadlineColor:     color.New(color.FgGreen),
	HelpHeadlineUnderline: true,
	HelpSubCommands:       true,

	Flags: func(f *grumble.Flags) {
		f.String("d", "directory", "DEFAULT", "set an alternative root directory path")
		f.Bool("v", "verbose", false, "enable verbose mode")
	},
})

func init() {
	global.Env = map[string]string{}

	App.SetPrintASCIILogo(func(a *grumble.App) {
		fmt.Println(` Data Migration Kit`)
		fmt.Println(`  ____  _____ _____ `)
		fmt.Println(` |    \|     |  |  |`)
		fmt.Println(` |  |  | | | |    -|`)
		fmt.Println(` |____/|_|_|_|__|__|`)
		fmt.Println(`              v0.0.1`)
		fmt.Println()
		fmt.Println(` type "help" for cmds`)
		fmt.Println()
	})

}
