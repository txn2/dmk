package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/AlecAivazis/survey"
	"github.com/cjimti/migration-kit/cfg"
	"github.com/desertbit/grumble"
	"github.com/go-yaml/yaml"
)

func init() {
	createCmd := &grumble.Command{
		Name:    "create",
		Help:    "create a projects, databases, queries and transformations",
		Aliases: []string{"add"},
	}

	App.AddCommand(createCmd)

	createCmd.AddCommand(&grumble.Command{
		Name:    "project",
		Help:    "create a project",
		Aliases: []string{"proj"},
		Run: func(c *grumble.Context) error {
			createProject()
			return nil
		},
	})

	createCmd.AddCommand(&grumble.Command{
		Name:    "database",
		Help:    "create a database",
		Aliases: []string{"db"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				createDatabase()
			}
			return nil
		},
	})

	createCmd.AddCommand(&grumble.Command{
		Name:    "query",
		Help:    "create a query",
		Aliases: []string{"q"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				createQuery()
			}
			return nil
		},
	})

	createCmd.AddCommand(&grumble.Command{
		Name:    "migration",
		Help:    "create a migration",
		Aliases: []string{"m"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				createMigration()
			}
			return nil
		},
	})

}

func activeProjectCheck() (ok bool) {
	if global.Project.Component.MachineName == "" {
		App.PrintError(errors.New("no active project"))
		return false
	}

	return true
}

func createDatabase() {
	name := ""
	prompt := &survey.Input{
		Message: "Database Name:",
		Help:    "Human readable name. Ex: `ACME Production`",
	}
	survey.AskOne(prompt, &name, nil)

	machineName := machineName(name)

	description := ""
	prompt = &survey.Input{
		Message: "Database Description:",
		Help:    "Ex: `ACME production mysql`",
	}
	survey.AskOne(prompt, &description, nil)

	component := cfg.Component{
		Kind:        "Database",
		MachineName: machineName,
		Name:        name,
		Description: description,
	}

	database := cfg.Database{
		Component: component,
	}

	if global.Project.Databases == nil {
		global.Project.Databases = map[string]cfg.Database{}
	}

	global.Project.Databases[machineName] = database
	saved := confirmAndSave(global.Project.Component.MachineName, global.Project)
	if saved {
		fmt.Println()
		fmt.Printf("NOTICE: Database %s was saved.\n", name)
	}
}

func createMigration() {
	name := ""
	prompt := &survey.Input{
		Message: "Migrate Name:",
		Help:    "Human readable name. Ex: `Migrate users`",
	}
	survey.AskOne(prompt, &name, nil)

	machineName := machineName(name)

	description := ""
	prompt = &survey.Input{
		Message: "Migration Description:",
		Help:    "Ex: `Migrate all users from the user table.`",
	}
	survey.AskOne(prompt, &description, nil)

	component := cfg.Component{
		Kind:        "Migration",
		MachineName: machineName,
		Name:        name,
		Description: description,
	}

	migration := cfg.Migration{
		Component: component,
	}

	// sources and destinations
	queries := make([]string, 0)

	for k := range global.Project.Queries {
		queries = append(queries, k)
	}

	dbs := make([]string, 0)

	for k := range global.Project.Databases {
		dbs = append(dbs, k)
	}

	promptSelect := &survey.Select{
		Message: "Choose a SOURCE Database:",
		Options: dbs,
	}
	survey.AskOne(promptSelect, &migration.SourceDb, nil)

	promptSelect = &survey.Select{
		Message: "Choose a SOURCE Query:",
		Options: queries,
	}
	survey.AskOne(promptSelect, &migration.SourceQuery, nil)

	promptSelect = &survey.Select{
		Message: "Choose a DESTINATION Database:",
		Options: dbs,
	}
	survey.AskOne(promptSelect, &migration.DestinationDb, nil)

	promptSelect = &survey.Select{
		Message: "Choose a DESTINATION Query:",
		Options: queries,
	}
	survey.AskOne(promptSelect, &migration.DestinationQuery, nil)

	if global.Project.Migrations == nil {
		global.Project.Migrations = map[string]cfg.Migration{}
	}

	global.Project.Migrations[machineName] = migration
	saved := confirmAndSave(global.Project.Component.MachineName, global.Project)
	if saved {
		fmt.Println()
		fmt.Printf("NOTICE: Migration %s was saved.\n", name)
	}
}

func createQuery() {
	name := ""
	prompt := &survey.Input{
		Message: "Query Name:",
		Help:    "Human readable name. Ex: `Get all users`",
	}
	survey.AskOne(prompt, &name, nil)

	machineName := machineName(name)

	description := ""
	prompt = &survey.Input{
		Message: "Query Description:",
		Help:    "Ex: `Get all users from the user table.`",
	}
	survey.AskOne(prompt, &description, nil)

	component := cfg.Component{
		Kind:        "Query",
		MachineName: machineName,
		Name:        name,
		Description: description,
	}

	statement := ""
	prompt = &survey.Input{
		Message: "Statement:",
		Help:    "Ex: `SELECT * FROM client_user;`",
	}
	survey.AskOne(prompt, &statement, nil)

	query := cfg.Query{
		Component: component,
		Statement: statement,
	}

	if global.Project.Queries == nil {
		global.Project.Queries = map[string]cfg.Query{}
	}

	global.Project.Queries[machineName] = query
	saved := confirmAndSave(global.Project.Component.MachineName, global.Project)
	if saved {
		fmt.Println()
		fmt.Printf("NOTICE: Query %s was saved.\n", name)
	}

}

func createProject() {

	name := ""
	prompt := &survey.Input{
		Message: "Project Name:",
	}
	survey.AskOne(prompt, &name, nil)

	machineName := machineName(name)

	description := ""
	prompt = &survey.Input{
		Message: "Project Description:",
	}
	survey.AskOne(prompt, &description, nil)

	component := cfg.Component{
		Kind:        "Project",
		MachineName: machineName,
		Name:        name,
		Description: description,
	}

	project := cfg.Project{
		Component: component,
	}

	saved := confirmAndSave(machineName, project)
	if saved {
		fmt.Println()
		fmt.Printf("NOTICE: Project %s was saved.\n", name)
		SetProject(project)
	}
}

func confirmAndSave(machineName string, component interface{}) bool {
	filename := machineName + "-mdk.yml"

	save := false
	saveMessage := fmt.Sprintf("Save project file %s?", filename)
	savePrompt := &survey.Confirm{
		Message: saveMessage,
	}
	survey.AskOne(savePrompt, &save, nil)

	if save == false {
		fmt.Println()
		fmt.Printf("NOTICE: %s was not saved.\n", filename)
		return false
	}

	if exists := fileExists(filename); exists != false {
		overMessage := fmt.Sprintf("WARNING: Project File %s exists. Overwrite?", filename)
		overPrompt := &survey.Confirm{
			Message: overMessage,
		}

		survey.AskOne(overPrompt, &save, nil)
	}

	if save == false {
		fmt.Println()
		fmt.Printf("NOTICE: %s was not saved.\n", filename)
		return false
	}

	// Marshal to YML and Save
	d, err := yaml.Marshal(component)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	err = ioutil.WriteFile("./"+filename, d, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	return true
}
