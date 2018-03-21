package cfg

import "github.com/cjimti/migration-kit/driver"

// Component is a generic key value set defining a component
type Component struct {
	Kind        string
	Name        string
	MachineName string `yaml:"machineName"`
	Description string
}

// Database defines a database and it's configuration
type Database struct {
	Component     Component
	Driver        string
	Configuration driver.Config
}

// Migration defines a source and destination database,
// query and transformation script
type Migration struct {
	Component            Component
	SourceDb             string // db machine name
	DestinationDb        string // db machine name
	SourceQuery          string // how to get the data
	DestinationQuery     string // how to insert the data
	TransformationScript string // js script for specialized data processing
}

// Project defines an overall project consisting of
// Databases and Migrations
type Project struct {
	Component  Component
	Databases  map[string]Database  // map of database machine names to databases
	Migrations map[string]Migration // map or migration machine names to migrations
}
