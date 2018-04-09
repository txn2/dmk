package cli

import (
	"io/ioutil"
	"os"
	"regexp"

	"fmt"

	"github.com/cjimti/dmk/migrate"
	"github.com/desertbit/grumble"
	"github.com/olekukonko/tablewriter"
)

func init() {
	listCmd := &grumble.Command{
		Name:    "list",
		Help:    "list components such as projects, databases, and migrations",
		Aliases: []string{"ls"},
	}

	Cli.AddCommand(listCmd)

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
			if ok := activeProjectCheck(); ok {
				fmt.Printf("Drivers: %s\n", DriverManager.RegisteredDrivers())
			}
			return nil
		},
	})

	listCmd.AddCommand(&grumble.Command{
		Name:    "tunnels",
		Help:    "list tunnels",
		Aliases: []string{"t"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				listTunnels()
			}
			return nil
		},
	})

}

func listTunnels() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Tunnel", "Local >", "Server >", "Remote"})

	for k := range appState.Project.Tunnels {
		t := appState.Project.Tunnels[k]
		table.Append([]string{
			t.Component.MachineName,
			fmt.Sprintf("%s: %s", t.Component.Name, t.Component.Description),
			fmt.Sprintf("%s:%d", t.Local.Host, t.Local.Port),
			fmt.Sprintf("%s@%s:%d", t.TunnelAuth.User, t.Server.Host, t.Server.Port),
			fmt.Sprintf("%s:%d", t.Remote.Host, t.Remote.Port),
		})
	}

	table.Render()
}

func listMigrations() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Description", "Source DB", "Dest DB"})

	for k := range appState.Project.Migrations {
		m := appState.Project.Migrations[k]
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
	table.SetHeader([]string{"Machine Name", "Name", "Description", "Tunnel"})

	for k := range appState.Project.Databases {
		db := appState.Project.Databases[k]
		table.Append([]string{db.Component.MachineName, db.Component.Name, db.Component.Description, db.Tunnel})
	}

	table.Render()
	fmt.Println("Try \"desc db [MACHINE NAME]\" to describe a database.")
}

func listProjects() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Machine Name", "Project", "File Name", "Description"})

	projects, _ := GetProjects()

	for _, p := range projects {
		filename := appState.Directory + p.Component.MachineName + "-dmk.yml"
		table.Append([]string{p.Component.MachineName, p.Component.Name, filename, p.Component.Description})
	}

	table.Render()
	fmt.Println("Try \"open p [MACHINE NAME]\" to open a project.")
	fmt.Println("Try \"desc p [MACHINE NAME]\" to describe a project.")
}

// GetProjects returns an array slice of projects.
func GetProjects() (projects []migrate.Project, err error) {
	files, err := ioutil.ReadDir(appState.Directory)
	if err != nil {
		return projects, err
	}

	for _, f := range files {
		filename := f.Name()

		// if file ends with -mdk.yml it's a project file
		// load it and get the name
		match, _ := regexp.MatchString("-dmk\\.yml$", filename)

		if match {
			project, _ := migrate.LoadProject(appState.Directory + filename)
			projects = append(projects, project)
		}
	}

	return projects, nil
}
