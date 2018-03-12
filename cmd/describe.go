package cmd

import (
	"fmt"

	"os"

	"github.com/cjimti/migration-kit/cfg"
	"github.com/desertbit/grumble"
	"github.com/olekukonko/tablewriter"
)

func init() {
	listCmd := &grumble.Command{
		Name:    "describe",
		Help:    "describe components such as projects, databases, queries, transformations and migrations",
		Aliases: []string{"desc"},
	}

	App.AddCommand(listCmd)

	listCmd.AddCommand(&grumble.Command{
		Name:      "project",
		Help:      "describe project",
		Usage:     "describe project [machine_name]",
		Aliases:   []string{"proj"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if len(c.Args) == 1 {
				describeProject(c.Args[0])
				return nil
			}
			fmt.Printf("Try: %s\n", c.Command.Usage)
			return nil
		},
	})

}

func describeProject(machineName string) {
	p, err := loadProject(machineName + "-mdk.yml")
	if err != nil {
		App.PrintError(err)
		return
	}

	dbMachNames := make([]string, 0, len(p.Databases))
	for k := range p.Databases {
		dbMachNames = append(dbMachNames, k)
	}

	describeComponent(p.Component)

	table := tablewriter.NewWriter(os.Stdout)
	table.Append([]string{"Databases", fmt.Sprintf("%s", dbMachNames)})
	table.Append([]string{"Queries", string(len(p.Queries))})
	table.Append([]string{"Migrations", string(len(p.Migrations))})

	table.Render()
	fmt.Println()
}

func describeComponent(c cfg.Component) {

	fmt.Println()
	fmt.Printf("Compoent Kind: %s\n", c.Kind)
	fmt.Println()
	table := tablewriter.NewWriter(os.Stdout)
	table.Append([]string{"MachineName", c.MachineName})
	table.Append([]string{"Name", c.Name})
	table.Append([]string{"Description", c.Description})
	table.SetBorder(false)
	table.Render()
	fmt.Println()
}
