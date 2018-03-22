package cmd

import (
	"fmt"
	"io/ioutil"

	"strconv"

	"math/rand"

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
				createDatabase(cfg.Database{})
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

	createCmd.AddCommand(&grumble.Command{
		Name:    "tunnel",
		Help:    "create an ssh tunnel",
		Aliases: []string{"t"},
		Run: func(c *grumble.Context) error {
			if ok := activeProjectCheck(); ok {
				createTunnel()
			}
			return nil
		},
	})

}

func createTunnel() {
	name := ""
	namePrompt := &survey.Input{
		Message: "Tunnel Name:",
		Help:    "Human readable name. Ex: `ACME Production Server`",
	}
	survey.AskOne(namePrompt, &name, nil)

	machineName := machineName(name)

	description := ""
	descPrompt := &survey.Input{
		Message: "Tunnel Description:",
		Help:    "Ex: `ACME production server with localhost access to mysql.`",
	}
	survey.AskOne(descPrompt, &description, nil)

	component := cfg.Component{
		Kind:        "Tunnel",
		MachineName: machineName,
		Name:        name,
		Description: description,
	}

	fmt.Printf("Configure local endpoint (this machine):\n")

	randPort := strconv.Itoa(3000 + rand.Intn(3000))
	localEp, err := createEndpoint("Local", "localhost", randPort)
	if err != nil {
		App.PrintError(err)
		return
	}

	fmt.Printf("Configure server endpoint (tunnel to):\n")
	serverEp, err := createEndpoint("Server", "", "22")
	if err != nil {
		App.PrintError(err)
		return
	}

	authUser := ""
	authUserPrompt := &survey.Input{
		Message: "Server SSH Username",
		Help:    "Username used for server ssh connection.`",
	}
	survey.AskOne(authUserPrompt, &authUser, nil)

	fmt.Printf("Configure remote endpoint (destination):\n")
	remoteEp, err := createEndpoint("Remote", "localhost", "3306")
	if err != nil {
		App.PrintError(err)
		return
	}

	tunnelCfg := cfg.Tunnel{
		Component: component,
		Local:     localEp,
		Server:    serverEp,
		Remote:    remoteEp,
		TunnelAuth: cfg.TunnelAuth{
			User: authUser,
		},
	}

	if global.Project.Tunnels == nil {
		global.Project.Tunnels = map[string]cfg.Tunnel{}
	}

	global.Project.Tunnels[machineName] = tunnelCfg
	saved := confirmAndSave(global.Project.Component.MachineName, global.Project)
	if saved {
		fmt.Println()
		fmt.Printf("NOTICE: Tunnel %s was saved.\n", name)
	}

}

func createEndpoint(name string, defH string, defP string) (cfg.Endpoint, error) {

	host := ""
	hostPrompt := &survey.Input{
		Message: name + " Host:",
		Default: defH,
	}
	survey.AskOne(hostPrompt, &host, nil)

	port := ""
	portPrompt := &survey.Input{
		Message: name + " Port:",
		Default: defP,
	}
	survey.AskOne(portPrompt, &port, nil)

	portN, err := strconv.Atoi(port)
	if err != nil {
		return cfg.Endpoint{}, err
	}

	endpoint := cfg.Endpoint{
		Host: host,
		Port: portN,
	}

	return endpoint, nil
}

func createDatabase(database cfg.Database) {
	name := ""
	prompt := &survey.Input{
		Message: "Database Name:",
		Help:    "Human readable name. Ex: `ACME Production`",
		Default: database.Component.Name,
	}
	survey.AskOne(prompt, &name, nil)

	if database.Component.Name != "" {
		name = database.Component.Name
	}
	machineName := machineName(name)

	description := ""
	prompt = &survey.Input{
		Message: "Database Description:",
		Help:    "Ex: `ACME production mysql`",
		Default: database.Component.Description,
	}
	survey.AskOne(prompt, &description, nil)

	component := cfg.Component{
		Kind:        "Database",
		MachineName: machineName,
		Name:        name,
		Description: description,
	}

	useTunnel := false
	if database.Tunnel != "" {
		useTunnel = true
	}

	useTunnelPrompt := &survey.Confirm{
		Message: "Does database require a tunnel?",
		Default: useTunnel,
	}
	survey.AskOne(useTunnelPrompt, &useTunnel, nil)

	if useTunnel {
		// get tunnel list
		tunnels := make([]string, 0)

		for k := range global.Project.Tunnels {
			tunnels = append(tunnels, k)
		}

		tunnelPrompt := &survey.Select{
			Message: "Choose a tunnel:",
			Options: tunnels,
		}
		survey.AskOne(tunnelPrompt, &database.Tunnel, nil)
	}

	// add component to database
	database.Component = component
	// configure the database
	promptSelect := &survey.Select{
		Message: "Choose a database driver:",
		Options: DriverManager.RegisteredDrivers(),
		Default: database.Driver,
	}
	survey.AskOne(promptSelect, &database.Driver, nil)

	// configure the driver
	dbDriver, err := DriverManager.GetNewDriver(database.Driver)
	if err != nil {
		App.PrintError(err)
	}

	if database.Configuration == nil {
		database.Configuration = driver.Config{}
	}

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

	err = ioutil.WriteFile(global.Directory+filename, d, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	return true
}
