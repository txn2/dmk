package driver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/AlecAivazis/survey"
	"github.com/recursionpharma/go-csv-map"
)

// CSV implements data.Driver
type CSV struct {
	config Config
}

// ArgCount calculate the number of expected arguments for
// a specified query with this driver.
func (c *CSV) ArgCount(query string) int {
	return 0
}

// Init initializes at the beginning of each run.
func (c *CSV) Init() {

}

// HasOutQuery is false for CSV
func (c *CSV) HasOutQuery() bool {
	return false
}

// HasInQuery is false for CSV
func (c *CSV) HasInQuery() bool {
	return false
}

// HasCountQuery is false for CSV
// TODO: implement counter
func (c *CSV) HasCountQuery() bool {
	return false
}

// Configure (keys determined in ConfigSurvey)
func (c *CSV) Configure(config Config) error {

	// Validation
	_, ok := config["filePath"]
	if ok != true {
		return errors.New("missing config key filePath")
	}

	c.config = config

	return nil
}

// Done for Driver interface.
func (c *CSV) Done() error {
	return nil
}

// In for Driver interface. @TODO implementation
func (c *CSV) In(query string, args []string, record Record) error {
	fmt.Printf("CSV In is not yet implemented.\n")
	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
// false value mean indefinite.
// TODO: Implement expected out for CSV
func (c *CSV) ExpectedOut() (bool, int, error) {
	return false, 0, nil
}

// Out for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (c *CSV) Out(query string, args []string) (<-chan Record, error) {
	// call Configure with a driver.Config first
	if c.config == nil {
		return nil, errors.New("CSV is not configured")
	}

	recordChan := make(chan Record, 1)

	if filePathC, ok := c.config["filePath"]; ok {
		filePath, ok := filePathC.(string) // type assertion
		if ok != true {
			return nil, errors.New("configured value of CSV filePath is not a string")
		}

		csvIn, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		r := csvmap.NewReader(csvIn)
		r.Columns, err = r.ReadHeader()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		go func() {
			for {
				rec, err := r.Read()
				if err == io.EOF {
					close(recordChan)
					break
				}
				if err != nil {
					log.Fatal(err)
				}

				// convert csv record to Record
				record := Record{}
				for key, value := range rec {
					record[key] = value
				}

				// send the record out the channel
				recordChan <- record
			}
		}()

	}

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (c *CSV) ConfigSurvey(config Config, machineName string) error {
	fmt.Println("---- CSV Driver Configuration ----")

	filePath := ""
	prompt := &survey.Input{
		Message: "File:",
		Help:    "Path to CSV file: \"./somedir/somefile.csv\"",
	}
	survey.AskOne(prompt, &filePath, nil)
	config["filePath"] = filePath

	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("csv", func() Driver { return new(CSV) })
}
