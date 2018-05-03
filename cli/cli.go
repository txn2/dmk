package cli

import (
	"fmt"
	"io/ioutil"
	"os"

	"errors"

	"github.com/AlecAivazis/survey"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"github.com/go-yaml/yaml"
	"github.com/txn2/dmk/driver"
	"github.com/txn2/dmk/migrate"
	"github.com/txn2/dmk/tunnel"
)

// appState holds state for the CLI
var appState struct {
	Project   migrate.Project
	Directory string // base directory for commandline interaction with projects
	Env       map[string]string
}

var Version = "0.0.0"

// Cli is the DMK CLI
// see http://github.com/desertbit/grumble
var Cli = grumble.New(&grumble.Config{
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

// TunnelManager manages the available tunnels.
var TunnelManager = tunnel.Manager{}

// init the cmd package
func init() {
	appState.Env = make(map[string]string, 0)

	Cli.SetPrintASCIILogo(func(a *grumble.App) {
		fmt.Println(` Data Migration Kit`)
		fmt.Println(`  ___  _____ _____ `)
		fmt.Println(` |   \|     |  |  |`)
		fmt.Println(` | |  | | | |    -|`)
		fmt.Println(` |___/|_|_|_|__|__|`)
		fmt.Println(`             v` + Version)
		fmt.Println()
		fmt.Println(` type "help" for cmds`)
		fmt.Println(` type "ls p" for a list of projects`)
		fmt.Println(` type "create p" to create a project`)
		fmt.Println()
	})

	Cli.OnInit(func(a *grumble.App, flags grumble.FlagMap) error {
		// Set the base directory
		dir, err := prepDirectory(flags.String("directory"))
		if err != nil {
			Cli.PrintError(errors.New("directory " + dir + " does not exist"))
			os.Exit(1)
		}

		appState.Directory = dir

		Cli.SetPrompt("dmk [" + appState.Directory + "] » ")

		// Get the default project from the command line if specified
		project := flags.String("project")
		if project != "" {
			a.RunCommand([]string{"open", "p", flags.String("project")})
		}
		return nil
	})

}

// prepDirectory adds a slash if one does not exist and checks the
// existence of the directory it.
func prepDirectory(dir string) (d string, err error) {

	// should end in a slash, if not add one
	if dir[len(dir)-1:] != "/" {
		dir = dir + "/"
	}

	if _, err := ioutil.ReadDir(dir); err != nil {
		return dir, err
	}

	return dir, nil
}

// SetProject sets a project as the active project
func SetProject(project migrate.Project) {
	appState.Project = project
	Cli.SetPrompt("dmk [" + appState.Directory + "] » " + project.Component.MachineName + " » ")
}

// activeProjectCheck is a simple check to see if we have a current
// active project.
func activeProjectCheck() (ok bool) {
	if appState.Project.Component.MachineName == "" {
		Cli.PrintError(errors.New("no active project"))
		fmt.Println("Try \"ls p\" for a list of projects.")
		return false
	}

	return true
}

// fileExists checks for the existence of a file
func fileExists(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}

// PromptCfg configures the visual prompt.
type PromptCfg struct {
	Value   *string
	Message string
	Help    string
	Default string
}

func dbChooser(pcfg PromptCfg) {
	dbs := make([]string, 0)

	for k := range appState.Project.Databases {
		dbs = append(dbs, k)
	}

	sourceDbPrompt := &survey.Select{
		Message: pcfg.Message + ":",
		Help:    pcfg.Help,
		Default: pcfg.Default,
		Options: dbs,
	}
	survey.AskOne(sourceDbPrompt, pcfg.Value, nil)
}

// confirmAndSave prompts a user before a save.
func confirmAndSave(machineName string, component interface{}) bool {
	filename := machineName + "-dmk.yml"

	save := false
	saveMessage := fmt.Sprintf("Save project file %s?", filename)
	savePrompt := &survey.Confirm{
		Message: saveMessage,
	}
	survey.AskOne(savePrompt, &save, nil)

	if save == false {
		fmt.Println()
		fmt.Printf("NOTICE: %s was not saved.\n", filename)
		return false
	}

	if exists := fileExists(filename); exists != false {
		overMessage := fmt.Sprintf("WARNING: Project File %s exists. Overwrite?", filename)
		overPrompt := &survey.Confirm{
			Message: overMessage,
		}

		survey.AskOne(overPrompt, &save, nil)
	}

	if save == false {
		fmt.Println()
		fmt.Printf("NOTICE: %s was not saved.\n", filename)
		return false
	}

	// Marshal to YML and Save
	d, err := yaml.Marshal(component)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	err = ioutil.WriteFile(appState.Directory+filename, d, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	return true
}

func fileLog(machineName string) (*os.File, error) {

	pMn := appState.Project.Component.MachineName
	f, err := os.Create(appState.Directory + pMn + "-" + machineName + ".log")

	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return f, err
}
