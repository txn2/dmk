package cfg

import (
	"github.com/cjimti/dmk/driver"
)

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
	Driver        string // driver type
	Tunnel        string // tunnel machine name
	Configuration driver.Config
}

// Migration defines a source and destination database,
// query and transformation script
type Migration struct {
	Component             Component
	SourceDb              string `yaml:"sourceDb"`              // db machine name
	DestinationDb         string `yaml:"destinationDb"`         // db machine name
	SourceQuery           string `yaml:"sourceQuery"`           // how to get the data
	SourceQueryNArgs      int    `yaml:"sourceQueryNArgs"`      // number of argument the source query takes
	SourceCountQuery      string `yaml:"sourceCountQuery"`      // for drivers that can count
	DestinationQuery      string `yaml:"destinationQuery"`      // how to insert the data
	DestinationQueryNArgs int    `yaml:"destinationQueryNArgs"` // number of arguments the destination query takes
	TransformationScript  string `yaml:"transformationScript"`  // js script for specialized data processing
}

// TunnelAuth defines tunnel authentication method
// TODO: Support ssh keys (currently only ssh agent)
type TunnelAuth struct {
	User string
}

// Endpoint contains a host and port tunnel endpoint.
type Endpoint struct {
	Host string
	Port int
}

// Tunnel defines an ssh tunnel
type Tunnel struct {
	Component  Component
	Local      Endpoint
	Server     Endpoint
	Remote     Endpoint
	TunnelAuth TunnelAuth `yaml:"tunnelAuth"`
}
