package cli

import (
	"fmt"

	"github.com/cjimti/migration-kit/migrate"
	"github.com/desertbit/grumble"
)

func init() {
	listCmd := &grumble.Command{
		Name:    "open",
		Help:    "open components such as projects, databases, queries, transformations and migrations",
		Aliases: []string{"o"},
	}

	App.AddCommand(listCmd)

	listCmd.AddCommand(&grumble.Command{
		Name:      "project",
		Help:      "open project",
		Usage:     "open project [machine_name]",
		Aliases:   []string{"p", "proj"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if len(c.Args) == 1 {
				openProject(c.Args[0])
				return nil
			}
			fmt.Printf("Try: %s\n", c.Command.Usage)
			return nil
		},
	})

}

// openProject by machine name
func openProject(machineName string) {
	project, err := migrate.LoadProject(appState.Directory + machineName + "-dmk.yml")
	if err != nil {
		App.PrintError(err)
		return
	}

	SetProject(project)
}
