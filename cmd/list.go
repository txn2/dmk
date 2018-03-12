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
		Aliases: []string{"proj"},
		Run: func(c *grumble.Context) error {
			listProjects()
			return nil
		},
	})

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
