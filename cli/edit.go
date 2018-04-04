package cli

import (
	"fmt"

	"errors"

	"github.com/AlecAivazis/survey"
	"github.com/desertbit/grumble"
)

func init() {
	editCmd := &grumble.Command{
		Name:    "edit",
		Help:    "edit databases",
		Aliases: []string{"e"},
	}

	Cli.AddCommand(editCmd)

	editCmd.AddCommand(&grumble.Command{
		Name:      "database",
		Help:      "edit a database",
		Aliases:   []string{"db", "d"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {

				if len(c.Args) == 1 {
					editDatabase(c.Args[0])
					return nil
				}

				dbs := make([]string, 0)

				for k := range appState.Project.Databases {
					dbs = append(dbs, k)
				}

				dbMachineName := ""
				dbPrompt := &survey.Select{
					Message: "Choose a database to edit:",
					Options: dbs,
				}
				survey.AskOne(dbPrompt, &dbMachineName, nil)

				editDatabase(dbMachineName)

			}
			return nil
		},
	})

}

func editDatabase(machineName string) {
	fmt.Printf("Edit database %s\n", machineName)

	if database, ok := appState.Project.Databases[machineName]; ok {
		createDatabase(database)
		return
	}

	Cli.PrintError(errors.New("can't find database: " + machineName))
}
