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
		Run: func(c *grumble.Context) error {
			if global.Project.Component.MachineName != "" {
				describeProject(global.Project.Component.MachineName)
				return nil
			}
			fmt.Println("No active project or project specified.")
			fmt.Println("Try \"desc PROJECT\" or \"list projects\"")
			return nil
		},
	}

	App.AddCommand(listCmd)

	listCmd.AddCommand(&grumble.Command{
		Name:      "project",
		Help:      "describe a project",
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

	listCmd.AddCommand(&grumble.Command{
		Name:      "query",
		Help:      "describe a query",
		Usage:     "describe query [machine_name]",
		Aliases:   []string{"q"},
		AllowArgs: true,
		Run: func(c *grumble.Context) error {
			if len(c.Args) == 1 {

				if ok := activeProjectCheck(); ok {
					describeQuery(c.Args[0])
				}

				return nil
			}
			fmt.Printf("Try: %s\n", c.Command.Usage)
			return nil
		},
	})
}

func describeQuery(machineName string) {
	if q, ok := global.Project.Queries[machineName]; ok {
		describeComponent(q.Component)
		fmt.Println("Statement:")
		fmt.Println()
		fmt.Printf("\t%s\n", q.Statement)
		fmt.Println()
		return
	}

	fmt.Println("Can not find query: " + machineName)
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

	qMachNames := make([]string, 0, len(p.Queries))
	for k := range p.Queries {
		qMachNames = append(qMachNames, k)
	}

	mMachNames := make([]string, 0, len(p.Migrations))
	for k := range p.Migrations {
		mMachNames = append(mMachNames, k)
	}

	describeComponent(p.Component)

	table := tablewriter.NewWriter(os.Stdout)
	table.Append([]string{"Databases", listing(dbMachNames)})
	table.Append([]string{"Queries", listing(qMachNames)})
	table.Append([]string{"Migrations", listing(mMachNames)})

	table.Render()
	fmt.Println()
}

func listing(keys []string) (tlist string) {
	first := true
	for _, k := range keys {
		if first != true {
			tlist = tlist + "\n"
		}
		tlist = tlist + fmt.Sprintf("%s", k)
		first = false
	}
	return tlist
}

func describeComponent(c cfg.Component) {

	fmt.Println()
	fmt.Printf("Component Kind: %s\n", c.Kind)
	fmt.Println()
	table := tablewriter.NewWriter(os.Stdout)
	table.Append([]string{"MachineName", c.MachineName})
	table.Append([]string{"Name", c.Name})
	table.Append([]string{"Description", c.Description})
	table.SetBorder(false)
	table.Render()
	fmt.Println()
}
