package cmd

import (
	"fmt"
	"os"

	"log"
	"regexp"
	"strings"

	"io/ioutil"

	"errors"

	"github.com/AlecAivazis/survey"
	"github.com/cjimti/migration-kit/cfg"
	"github.com/cjimti/migration-kit/driver"
	"github.com/cjimti/migration-kit/tunnel"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"github.com/go-yaml/yaml"
)

var global struct {
	Project   cfg.Project
	Directory string // base directory for commandline interaction with projects
	Env       map[string]string
}

// App is the DMK CLI
// see http://github.com/desertbit/grumble
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
		f.String("p", "project", "", "specify a project")
		f.String("d", "directory", "./", "specify a directory")
	},
})

// DriverManager manages the available database drivers.
var DriverManager = driver.DriverManager
var TunnelManager = tunnel.Manager{}

// init the cmd package
func init() {
	global.Env = make(map[string]string, 0)

	App.SetPrintASCIILogo(func(a *grumble.App) {
		fmt.Println(` Data Migration Kit`)
		fmt.Println(`  ___  _____ _____ `)
		fmt.Println(` |   \|     |  |  |`)
		fmt.Println(` | |  | | | |    -|`)
		fmt.Println(` |___/|_|_|_|__|__|`)
		fmt.Println(`             v0.0.1`)
		fmt.Println()
		fmt.Println(` type "help" for cmds`)
		fmt.Println(` type "ls p" for a list of projects`)
		fmt.Println(` type "create p" to create a project`)
		fmt.Println()
	})

	App.OnInit(func(a *grumble.App, flags grumble.FlagMap) error {
		// Set the base directory
		global.Directory = flags.String("directory")
		App.SetPrompt("dmk [" + global.Directory + "] » ")

		// Get the default project from the command line if specified
		project := flags.String("project")
		if project != "" {
			a.RunCommand([]string{"open", "p", flags.String("project")})
		}
		return nil
	})
}

// loadProject loads a project from yaml data
func loadProject(filename string) (project cfg.Project, err error) {
	ymlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return project, err
	}

	project = cfg.Project{}

	err = yaml.Unmarshal([]byte(ymlData), &project)
	if err != nil {
		return project, err
	}

	return project, nil
}

// SetProject sets a project as the active project
func SetProject(project cfg.Project) {
	global.Project = project
	App.SetPrompt("dmk [" + global.Directory + "] » " + project.Component.MachineName + " » ")
}

// activeProjectCheck is a simple check to see if we have a current
// active project.
func activeProjectCheck() (ok bool) {
	if global.Project.Component.MachineName == "" {
		App.PrintError(errors.New("no active project"))
		fmt.Println("Try \"ls p\" for a list of projects.")
		return false
	}

	return true
}

// machineName makes a string with unsafe characters replaced
func machineName(name string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}

	machineName := strings.ToLower(reg.ReplaceAllString(name, "_"))

	prompt := &survey.Input{
		Message: "Machine Name:",
		Help: "\n The Machine Name is used for file names and referencing components." +
			"\n This should not contain spaces or special characters other than - and _" +
			"\n The Default should be acceptable.",
		Default: machineName,
	}

	survey.AskOne(prompt, &machineName, nil)

	return machineName
}

// fileExists checks for the existence of a file
func fileExists(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}
