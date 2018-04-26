package cli

import (
	"fmt"

	"github.com/AlecAivazis/survey"
	"github.com/txn2/dmk/migrate"
	"github.com/desertbit/grumble"
)

func init() {
	runCmd := &grumble.Command{
		Name:    "reload",
		Help:    "reload active project",
		Usage:   "reload",
		Aliases: []string{"rl"},
		Flags: func(f *grumble.Flags) {
			f.Bool("f", "force", false, "Don't ask for confirmation.")
		},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				force := c.Flags.Bool("force")

				reloadProject(force)
			}
			return nil
		},
	}

	Cli.AddCommand(runCmd)

}

// relodProject reloads active project
func reloadProject(force bool) {
	machineName := appState.Project.Component.MachineName
	projectName := appState.Project.Component.Name
	file := appState.Directory + appState.Project.Component.MachineName + "-dmk.yml"

	load := force

	if force == false {
		loadMessage := fmt.Sprintf("Reload %s project [%s] from %s?", projectName, machineName, file)
		loadPrompt := &survey.Confirm{
			Message: loadMessage,
			Help:    "Reloading a project will overwrite any unsaved changes.\nYou want to do this if you opted to not save after a create or edit.",
		}
		survey.AskOne(loadPrompt, &load, nil)
	}

	if load {
		project, err := migrate.LoadProject(file)
		if err != nil {
			Cli.PrintError(err)
			return
		}

		fmt.Printf("%s project [%s] reloaded from %s.\n", projectName, machineName, file)
		SetProject(project)

	}
}
