package driver

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/AlecAivazis/survey"
)

// CSV implements data.Driver
type CSV struct {
	config Config
}

// Execute for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (c *CSV) Execute(query string, args Args) (chan Record, error) {
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

		r := csv.NewReader(csvIn)
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(record)
		}

	}

	return recordChan, nil
}

func (c *CSV) PopulateConfig(config Config) error {
	fmt.Println("---- CSV Driver Configuration ----")

	filePath := ""
	prompt := &survey.Input{
		Message: "File:",
		Help:    "Path to CSV file: \"./somedir/somefile.csv\"",
	}
	survey.AskOne(prompt, &filePath, nil)
	config["filePath"] = filePath

	// populate
	c.config = config
	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("csv", new(CSV))
}
