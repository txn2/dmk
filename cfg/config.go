package cfg

import "github.com/cjimti/migration-kit/driver"

type Component struct {
	Kind        string
	Name        string
	MachineName string `yaml:"machineName"`
	Description string
}

type Database struct {
	Component     Component
	Driver        string
	Configuration driver.Config
}

type Migration struct {
	Component          Component
	SourceDb           string // db machine name
	DestinationDb      string // db machine name
	SourceQuery        string // how to get the data
	DestinationQuery   string // how to insert the data
	TransformationSpec interface{}
}

type Project struct {
	Component  Component
	Databases  map[string]Database  // map of database machine names to databases
	Migrations map[string]Migration // map or migration machine names to migrations
}

// Currently supported operations: shift, concat, coalesce, extract, timestamp, uuid, default, pass and delete
type TransformationSpec struct {
	Operation string
	Spec      map[string]interface{}
}
