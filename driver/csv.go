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

// In for Driver interface. @TODO implementation
func (c *CSV) In(query string) error {
	fmt.Printf("CSV In is not yet implemented.\n")
	return nil
}

// Out for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (c *CSV) Out(query string, args Args) (<-chan Record, error) {
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
func (c *CSV) ConfigSurvey(config Config) error {
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
