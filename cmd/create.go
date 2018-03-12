package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/AlecAivazis/survey"
	"github.com/cjimti/migration-kit/cfg"
	"github.com/desertbit/grumble"
	"github.com/go-yaml/yaml"
)

func init() {
	createCmd := &grumble.Command{
		Name: "create",
		Help: "create a projects, databases, queries and transformations",
	}

	App.AddCommand(createCmd)

	createCmd.AddCommand(&grumble.Command{
		Name: "project",
		Help: "create a project",
		Run: func(c *grumble.Context) error {
			createProject()
			return nil
		},
	})

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

	filename := machineName + "-mdk.yml"
	project := cfg.Project{
		Component: component,
	}

	saved := confirmAndSave(filename, project)
	if saved {
		fmt.Println()
		fmt.Printf("NOTICE: Project %s was saved as %s.\n", name, filename)
		SetProject(project)
	}
}

func confirmAndSave(filename string, component interface{}) bool {

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
