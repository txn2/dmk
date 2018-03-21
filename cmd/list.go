package cmd

import (
	"io/ioutil"
	"os"
	"regexp"

	"fmt"

	"github.com/cjimti/migration-kit/cfg"
	"github.com/desertbit/grumble"
	"github.com/olekukonko/tablewriter"
)

func init() {
	listCmd := &grumble.Command{
		Name:    "list",
		Help:    "list components such as projects, databases, and migrations",
		Aliases: []string{"ls"},
	}

	App.AddCommand(listCmd)

	listCmd.AddCommand(&grumble.Command{
		Name:    "projects",
		Help:    "list projects",
		Aliases: []string{"p"},
		Run: func(c *grumble.Context) error {
			listProjects()
			return nil
		},
	})

	listCmd.AddCommand(&grumble.Command{
		Name:    "databases",
		Help:    "list databases",
		Aliases: []string{"db", "d"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				listDatabases()
			}
			return nil
		},
	})

	listCmd.AddCommand(&grumble.Command{
		Name:    "migrations",
		Help:    "list migrations",
		Aliases: []string{"m"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				listMigrations()
			}
			return nil
		},
	})

	listCmd.AddCommand(&grumble.Command{
		Name: "drivers",
		Help: "list drivers",
		Run: func(c *grumble.Context) error {
			//			data.DriverManager.
			fmt.Printf("Drivers: %s\n", DriverManager.RegisteredDrivers())
			return nil
		},
	})

}

func listMigrations() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Description", "Source DB", "Dest DB"})

	for k := range global.Project.Migrations {
		m := global.Project.Migrations[k]
		table.Append([]string{
			m.Component.MachineName,
			fmt.Sprintf("%s: %s", m.Component.Name, m.Component.Description),
			m.SourceDb,
			m.DestinationDb,
		})
	}

	table.Render()
	fmt.Println("Try \"desc m [MACHINE NAME]\" to describe a migration.")
}

func listDatabases() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Name", "Description"})

	for k := range global.Project.Databases {
		db := global.Project.Databases[k]
		table.Append([]string{db.Component.MachineName, db.Component.Name, db.Component.Description})
	}

	table.Render()
	fmt.Println("Try \"desc db [MACHINE NAME]\" to describe a database.")
}

func listProjects() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Project", "File Name", "Description"})

	projects, _ := GetProjects()

	for _, p := range projects {
		filename := p.Component.MachineName + "-dmk.yml"
		table.Append([]string{p.Component.MachineName, p.Component.Name, filename, p.Component.Description})
	}

	table.Render()
	fmt.Println("Try \"open p [MACHINE NAME]\" to open a project.")
	fmt.Println("Try \"desc p [MACHINE NAME]\" to describe a project.")
}

// GetProjects returns an array slice of projects.
func GetProjects() (projects []cfg.Project, err error) {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		return projects, err
	}

	for _, f := range files {
		filename := f.Name()

		// if file ends with -mdk.yml it's a project file
		// load it and get the name
		match, _ := regexp.MatchString("-dmk\\.yml$", filename)

		if match {
			project, _ := loadProject(filename)
			projects = append(projects, project)
		}
	}

	return projects, nil
}
