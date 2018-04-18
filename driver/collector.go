package driver

import (
	"errors"
	"fmt"
)

// ResultCollectionItem represents a set of records and corresponding args.
type ResultCollectionItem struct {
	Record Record
	Args   []string
}

// ResultCollection represents a slice of ResultCollectionItem
type ResultCollection []ResultCollectionItem

// CollectorStore holds a map of ResultCollection
var CollectorStore = map[string]ResultCollection{}

// Collector implements data.Driver
// TODO: implement a generate persistent kv object store
type Collector struct {
	config        Config
	collectionKey string
	store         ResultCollection
	init          bool
}

// ArgCount calculate the number of expected arguments for
// a specified query with this driver.
func (c *Collector) ArgCount(query string) int {
	return 0
}

// Init initializes at the beginning of each run.
func (c *Collector) Init() {
	c.store = nil
}

// GetCollection returns slice of ResultCollectionItem
func (c *Collector) GetCollection() []ResultCollectionItem {
	return c.store
}

// HasOutQuery is false for Collector
func (c *Collector) HasOutQuery() bool {
	return true
}

// HasInQuery is false for Collector
func (c *Collector) HasInQuery() bool {
	return true
}

// HasCountQuery is false for Collector
func (c *Collector) HasCountQuery() bool {
	return false
}

// Configure (keys determined in ConfigSurvey)
func (c *Collector) Configure(config Config) error {
	c.config = config

	//fmt.Println("Configuring collector.")

	if _, ok := config["collectionKey"]; ok == false {
		fmt.Println("ERROR: no collectionKey")
		return errors.New("collectionKey does not exist in configuration")
	}

	c.collectionKey = c.config["collectionKey"].(string)

	return nil
}

// Done for Driver interface.
func (c *Collector) Done() error {
	return nil
}

// In for Driver interface.
func (c *Collector) In(query string, args []string, record Record) error {
	//fmt.Println("Got collector in.")
	// in the future query can be used to specify a different storage key and type
	rci := ResultCollectionItem{
		Record: record,
		Args:   args,
	}

	CollectorStore[c.collectionKey] = append(CollectorStore[c.collectionKey], rci)
	c.store = append(c.store, rci)

	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
func (c *Collector) ExpectedOut() (bool, int, error) {
	return true, len(CollectorStore[c.collectionKey]), nil
}

// Out for Driver interface.
func (c *Collector) Out(query string, args []string) (<-chan Record, error) {

	recordChan := make(chan Record, 1)

	go func() {
		if collection, ok := CollectorStore[c.collectionKey]; ok {
			for _, collectionItem := range collection {
				recordChan <- collectionItem.Record
			}
		}
		close(recordChan)
	}()

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (c *Collector) ConfigSurvey(config Config, machineName string) error {
	config["collectionKey"] = machineName
	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("collector", func() Driver { return new(Collector) })
}
