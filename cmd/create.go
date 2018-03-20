package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/AlecAivazis/survey"
	"github.com/cjimti/migration-kit/cfg"
	"github.com/cjimti/migration-kit/driver"
	"github.com/desertbit/grumble"
	"github.com/go-yaml/yaml"
)

func init() {
	createCmd := &grumble.Command{
		Name:    "create",
		Help:    "create projects, databases, and migrations",
		Aliases: []string{"add"},
	}

	App.AddCommand(createCmd)

	createCmd.AddCommand(&grumble.Command{
		Name:    "project",
		Help:    "create a project",
		Aliases: []string{"p"},
		Run: func(c *grumble.Context) error {
			createProject()
			return nil
		},
	})

	createCmd.AddCommand(&grumble.Command{
		Name:    "database",
		Help:    "create a database",
		Aliases: []string{"db", "d"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				createDatabase()
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

	// configure the database
	promptSelect := &survey.Select{
		Message: "Choose a database driver:",
		Options: DriverManager.RegisteredDrivers(),
	}
	survey.AskOne(promptSelect, &database.Driver, nil)

	// configure the driver
	dbDriver, err := DriverManager.GetNewDriver(database.Driver)
	if err != nil {
		App.PrintError(err)
	}

	database.Configuration = driver.Config{}

	// configuration survey
	dbDriver.ConfigSurvey(database.Configuration)

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

	dbs := make([]string, 0)

	for k := range global.Project.Databases {
		dbs = append(dbs, k)
	}

	sourceDbPrompt := &survey.Select{
		Message: "Choose a SOURCE Database:",
		Options: dbs,
	}
	survey.AskOne(sourceDbPrompt, &migration.SourceDb, nil)

	sourceQueryPrompt := &survey.Editor{
		Message: "SOURCE Query:",
		Help:    "Ex: `SELECT id,username FROM users`",
	}
	survey.AskOne(sourceQueryPrompt, &migration.SourceQuery, nil)

	script := false
	promptBool := &survey.Confirm{
		Message: "Does this data require a script for transformation?",
	}
	survey.AskOne(promptBool, &script, nil)

	if script == true {
		scriptPrompt := &survey.Editor{
			Message: "Javascript is sent an object named \"data\".",
			Help:    "Manipulate the \"data\" object with javascript",
		}
		survey.AskOne(scriptPrompt, &migration.TransformationScript, nil)
	}

	destDbPrompt := &survey.Select{
		Message: "Choose a DESTINATION Database:",
		Options: dbs,
	}
	survey.AskOne(destDbPrompt, &migration.DestinationDb, nil)

	dqPrompt := &survey.Editor{
		Message: "DESTINATION Query:",
		Help: `Ex: INSERT INTO table_name JSON '{"id": "{{.id"}}", "username": "{{.username"}}"}` +
			"\nsee: https://golang.org/pkg/text/template/",
	}
	survey.AskOne(dqPrompt, &migration.DestinationQuery, nil)

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
	filename := machineName + "-dmk.yml"

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
