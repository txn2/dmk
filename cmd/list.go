package cmd

import (
	"io/ioutil"
	"os"
	"regexp"

	"github.com/cjimti/migration-kit/cfg"
	"github.com/desertbit/grumble"
	"github.com/olekukonko/tablewriter"
)

func init() {
	listCmd := &grumble.Command{
		Name:    "list",
		Help:    "list components such as projects, databases, queries, transformations and migrations",
		Aliases: []string{"ls"},
	}

	App.AddCommand(listCmd)

	listCmd.AddCommand(&grumble.Command{
		Name:    "projects",
		Help:    "list projects",
		Aliases: []string{"proj", "projs"},
		Run: func(c *grumble.Context) error {
			listProjects()
			return nil
		},
	})

	listCmd.AddCommand(&grumble.Command{
		Name:    "databases",
		Help:    "list databases",
		Aliases: []string{"db", "dbs"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				listDatabases()
			}
			return nil
		},
	})

	listCmd.AddCommand(&grumble.Command{
		Name:    "queries",
		Help:    "list queries",
		Aliases: []string{"q", "qs", "query"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				listQueries()
			}
			return nil
		},
	})

}

func listDatabases() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Name", "Description"})

	for k := range global.Project.Databases {
		db := global.Project.Databases[k]
		table.Append([]string{db.Component.MachineName, db.Component.Name, db.Component.Description})
	}

	table.Render()

}

func listQueries() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Name", "Description", "Statement"})

	for k := range global.Project.Queries {
		q := global.Project.Queries[k]
		table.Append([]string{q.Component.MachineName, q.Component.Name, q.Component.Description, q.Statement})
	}

	table.Render()

}

func listProjects() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Project", "File Name", "Description"})

	projects, _ := GetProjects()

	for _, p := range projects {
		filename := p.Component.MachineName + "-mdk.yml"
		table.Append([]string{p.Component.MachineName, p.Component.Name, filename, p.Component.Description})
	}

	table.Render()

}

// GetDatabases returns an array slice of Database machine names
func GetDatabaseMachineNames(project cfg.Project) (dbMachineNames []string) {
	for k := range project.Databases {
		dbMachineNames = append(dbMachineNames, k)
	}

	return dbMachineNames
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
		match, _ := regexp.MatchString("-mdk\\.yml$", filename)

		if match {
			project, _ := loadProject(filename)
			projects = append(projects, project)
		}
	}

	return projects, nil
}
