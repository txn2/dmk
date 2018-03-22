package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey"
	"github.com/desertbit/grumble"
)

func init() {
	editCmd := &grumble.Command{
		Name:    "edit",
		Help:    "edit projects, databases, migrations and tunnels",
		Aliases: []string{"e"},
	}

	App.AddCommand(editCmd)

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

				for k := range global.Project.Databases {
					dbs = append(dbs, k)
				}

				dbMachineName := ""
				dbPrompt := &survey.Select{
					Message: "Choose a SOURCE Database:",
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
	fmt.Printf("Edit database %s", machineName)
}
