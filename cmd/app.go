package cmd

import (
	"fmt"
	"os"

	"log"
	"regexp"
	"strings"

	"io/ioutil"

	"github.com/AlecAivazis/survey"
	"github.com/cjimti/migration-kit/cfg"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"github.com/go-yaml/yaml"
)

var global struct {
	Project cfg.Project
	Env     map[string]string
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

func SetProject(project cfg.Project) {
	global.Project = project
	App.SetPrompt("dmk » " + project.Component.MachineName + " » ")
}

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

func fileExists(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}

	return false
}
