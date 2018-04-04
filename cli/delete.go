package cli

import (
	"fmt"

	"errors"

	"github.com/AlecAivazis/survey"
	"github.com/desertbit/grumble"
)

func init() {
	editCmd := &grumble.Command{
		Name:    "delete",
		Help:    "delete databases",
		Aliases: []string{"rm"},
	}

	Cli.AddCommand(editCmd)

	editCmd.AddCommand(&grumble.Command{
		Name:      "tunnel",
		Help:      "delete a tunnel",
		Aliases:   []string{"t"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {

				if len(c.Args) == 1 {
					deleteTunnel(c.Args[0])
					return nil
				}

				tunnels := make([]string, 0)

				for k := range appState.Project.Tunnels {
					tunnels = append(tunnels, k)
				}

				tunnelMachineName := ""
				tunnePrompt := &survey.Select{
					Message: "Choose a tunnel to remove:",
					Options: tunnels,
				}
				survey.AskOne(tunnePrompt, &tunnelMachineName, nil)

				deleteTunnel(tunnelMachineName)

			}
			return nil
		},
	})

	editCmd.AddCommand(&grumble.Command{
		Name:      "database",
		Help:      "delete a database",
		Aliases:   []string{"db", "d"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {

				if len(c.Args) == 1 {
					deleteDatabase(c.Args[0])
					return nil
				}

				dbs := make([]string, 0)

				for k := range appState.Project.Databases {
					dbs = append(dbs, k)
				}

				dbMachineName := ""
				dbPrompt := &survey.Select{
					Message: "Choose a database to remove:",
					Options: dbs,
				}
				survey.AskOne(dbPrompt, &dbMachineName, nil)

				deleteDatabase(dbMachineName)

			}
			return nil
		},
	})

}

func deleteTunnel(machineName string) {
	if _, ok := appState.Project.Tunnels[machineName]; ok {
		fmt.Printf("Removing tunnel %s.\n", machineName)
		delete(appState.Project.Tunnels, machineName)
		saved := confirmAndSave(appState.Project.Component.MachineName, appState.Project)
		if saved {
			fmt.Println()
			fmt.Printf("NOTICE: Project saved less tunnel %s.\n", machineName)
		}
		return
	}

	Cli.PrintError(errors.New("no tunnel found for " + machineName))
}

func deleteDatabase(machineName string) {
	if _, ok := appState.Project.Databases[machineName]; ok {
		fmt.Printf("Removing database %s.\n", machineName)
		delete(appState.Project.Databases, machineName)
		saved := confirmAndSave(appState.Project.Component.MachineName, appState.Project)
		if saved {
			fmt.Println()
			fmt.Printf("NOTICE: Project saved less database %s.\n", machineName)
		}
		return
	}

	Cli.PrintError(errors.New("no database found for " + machineName))
}
