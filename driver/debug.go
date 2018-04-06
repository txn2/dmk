package driver

import (
	"fmt"
)

// Debug implements data.Driver
type Debug struct {
	config Config
}

// HasOutQuery is false for Debug
func (d *Debug) HasOutQuery() bool {
	return true
}

// HasInQuery is false for Debug
func (d *Debug) HasInQuery() bool {
	return true
}

// HasCountQuery is false for Debug
func (d *Debug) HasCountQuery() bool {
	return false
}

// Configure (keys determined in ConfigSurvey)
func (d *Debug) Configure(config Config) error {
	d.config = config

	return nil
}

// Done for Driver interface.
func (d *Debug) Done() error {
	return nil
}

// In for Driver interface.
func (d *Debug) In(query string, args []string, record Record) error {
	fmt.Printf("-- Debug In -- \n")
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Args: In:\n")

	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
// false value mean indefinite.
func (d *Debug) ExpectedOut() (bool, int, error) {
	return false, 0, nil
}

// Out for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (d *Debug) Out(query string, args []string) (<-chan Record, error) {
	// call Configure with a driver.Config first

	recordChan := make(chan Record, 1)
	defer close(recordChan)

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (d *Debug) ConfigSurvey(config Config, machineName string) error {
	fmt.Println("---- No Debug Driver Configuration ----")

	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("debug", func() Driver { return new(Debug) })
}
