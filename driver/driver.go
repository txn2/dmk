package driver

import "errors"

// Config is a map or configuration data specific to a specialized Driver
type Config map[string]interface{}

// Record is a map of a single database record
//
type Record map[string]interface{}

// Get a value from a Record
func (r Record) Get(key string) interface{} {
	return r[key]
}

// Set a value on a Record
func (r Record) Set(key string, value interface{}) {
	r[key] = value
}

// Args are used for populating a query
//
type Args map[string]interface{}

// Get a value from Args
func (r Args) Get(key string) interface{} {
	return r[key]
}

// Set a value on Args
func (r Args) Set(key string, value interface{}) {
	r[key] = value
}

// DataMap is the minimal interface for Records and Args.
//
type DataMap interface {
	Get(key string) interface{}
	Set(key string, value interface{})
}

// Driver managed configuration and of a database and executes queries against it.
type Driver interface {
	Configure(config Config) error                          // Takes a config map
	ConfigSurvey(config Config, machineName string) error   // Interactive config generator
	Out(query string, args []string) (<-chan Record, error) // outbound data
	In(query string, args []string, record Record) error    // inbound data
	Done() error                                            // finalization tasks when runner is done with In
	// ExpectedOut is the number of records we expect
	// from the source, some drivers can determine
	// expected output without a source query
	ExpectedOut() (bool, int, error) // a false return means indefinite
	HasOutQuery() bool               // does this driver use a query to get data
	HasInQuery() bool                // does this driver use a query to set data
	HasCountQuery() bool             // does this driver have a count query
}

// Manager handles the collection of drivers
type Manager struct {
	// a map of of machine names to drivers
	drivers map[string]func() Driver
}

// AddDriver adds a driver to the DriverManager
func (m *Manager) AddDriver(machineName string, driverFactory func() Driver) {
	m.drivers[machineName] = driverFactory
}

// RegisteredDrivers returns a string slice of driver machine names
func (m *Manager) RegisteredDrivers() []string {
	drivers := make([]string, 0)

	for k := range m.drivers {
		drivers = append(drivers, k)
	}

	return drivers
}

// GetNewDriver returns a new un-configured Driver
func (m *Manager) GetNewDriver(machineName string) (Driver, error) {
	if driverFactory, ok := m.drivers[machineName]; ok {
		return driverFactory(), nil
	}

	return nil, errors.New("No such driver: " + machineName)
}

// NewManager creates a new driver manager
func NewManager() *Manager {
	return &Manager{
		drivers: make(map[string]func() Driver),
	}
}

// DriverManager is where drivers register.
var DriverManager = NewManager()
