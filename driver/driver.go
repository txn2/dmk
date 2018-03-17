package driver

import "errors"

type Record map[string]interface{}

type Args map[string]interface{}

type Config map[string]interface{}

type Driver interface {
	PopulateConfig(config Config) error
	Execute(query string, args Args) (chan Record, error)
}

// Manager types
type Manager struct {
	// a map of of machine names to drivers
	drivers map[string]Driver
}

// AddDriver adds a driver to the DriverManager
func (m *Manager) AddDriver(machineName string, driver Driver) {
	m.drivers[machineName] = driver
}

// RegisteredDrivers returns a string slice of driver machine names
func (m *Manager) RegisteredDrivers() []string {
	drivers := make([]string, 0)

	for k := range m.drivers {
		drivers = append(drivers, k)
	}

	return drivers
}

func (m *Manager) GetDriver(machineName string) (Driver, error) {
	if driver, ok := m.drivers[machineName]; ok {
		return driver, nil
	}

	return nil, errors.New("No such driver: " + machineName)
}

// NewManager creates a new driver manager
func NewManager() *Manager {
	return &Manager{
		drivers: make(map[string]Driver),
	}
}

// where drivers register
var DriverManager = NewManager()
