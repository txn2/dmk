package driver

import (
	"errors"
	"fmt"

	"strings"

	"github.com/AlecAivazis/survey"
)

// Argset implements data.Driver
type Argset struct {
	config Config
}

// ArgCount calculate the number of expected arguments for
// a specified query with this driver.
func (a *Argset) ArgCount(query string) int {
	return len(a.GetArgs())
}

// GetArgs returns the args as a string slice.
func (a *Argset) GetArgs() []string {

	if argConfig, ok := a.config["args"].([]interface{}); ok {
		argSet := make([]string, 0)

		// convert interfaces to strings
		for _, ag := range argConfig {
			argSet = append(argSet, ag.(string))
		}

		return argSet
	}

	return []string{}
}

// Init initializes at the beginning of each run.
func (a *Argset) Init() {

}

// HasOutQuery
func (a *Argset) HasOutQuery() bool {
	return false
}

// HasInQuery
func (a *Argset) HasInQuery() bool {
	return false
}

// HasCountQuery
func (a *Argset) HasCountQuery() bool {
	return false
}

// Configure (keys determined in ConfigSurvey)
func (a *Argset) Configure(config Config) error {

	// Validation
	_, ok := config["args"]
	if ok != true {
		return errors.New("missing config key args")
	}

	a.config = config

	return nil
}

// Done for Driver interface.
func (a *Argset) Done() error {
	return nil
}

// In for Driver interface. @TODO implementation
func (a *Argset) In(query string, args []string, record Record) error {
	fmt.Printf("Argset In is not yet implemented.\n")
	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
// false value mean indefinite.
func (a *Argset) ExpectedOut() (bool, int, error) {
	return false, 0, nil
}

// Out for Driver interface. Argset turns args into Record
func (a *Argset) Out(query string, args []string) (<-chan Record, error) {
	//call Configure with a driver.Config first
	if a.config == nil {
		return nil, errors.New("argset is not configured")
	}

	recordChan := make(chan Record, 1)

	record := Record{}

	argSet := a.GetArgs()

	for i, a := range argSet {
		if i < len(args) {
			record.Set(a, args[i])
		}
	}

	recordChan <- record
	close(recordChan)

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (a *Argset) ConfigSurvey(config Config, machineName string) error {
	fmt.Println("---- Argset Driver Configuration ----")

	args := ""
	prompt := &survey.Input{
		Message: "Named Arguments (Comma separated):",
		Help:    "Example: Days, Limit",
	}
	survey.AskOne(prompt, &args, nil)

	argSlice := strings.Split(args, ",")
	cleanArgs := make([]string, 0)

	for _, arg := range argSlice {
		cleanArgs = append(cleanArgs, strings.TrimSpace(arg))
	}

	config["args"] = cleanArgs

	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("argset", func() Driver { return new(Argset) })
}
