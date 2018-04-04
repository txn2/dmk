package migrate

import (
	"io/ioutil"

	"github.com/cjimti/migration-kit/cfg"
	"github.com/cjimti/migration-kit/driver"
	"github.com/cjimti/migration-kit/tunnel"
	"gopkg.in/yaml.v2"
)

// Project defines an overall project consisting of
// Databases and Migrations
type Project struct {
	Component     cfg.Component
	Databases     map[string]cfg.Database  // map of database machine names to databases
	Migrations    map[string]cfg.Migration // map of migration machine names to migrations
	Tunnels       map[string]cfg.Tunnel    // map of tunnels
	driverManager driver.Manager
	tunnelManager tunnel.Manager
}

// LoadProject loads a project from yaml data
func LoadProject(filename string) (project Project, err error) {
	ymlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return project, err
	}

	project = Project{}

	err = yaml.Unmarshal([]byte(ymlData), &project)
	if err != nil {
		return project, err
	}

	return project, nil
}
