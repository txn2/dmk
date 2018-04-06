package driver

import (
	"errors"
)

type ResultCollectionItem struct {
	Record *Record
	Args   *[]string
}

type ResultCollection []ResultCollectionItem

var CollectorStore = map[string]ResultCollection{}

// Collector implements data.Driver
// TODO: implement a generate persistent kv object store
type Collector struct {
	config        Config
	collectionKey string
}

// GetCollection returns slice of ResultCollectionItem
func (c *Collector) GetCollection() []ResultCollectionItem {
	if collection, ok := CollectorStore[c.collectionKey]; ok {
		return collection
	}

	return []ResultCollectionItem{}
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

	if _, ok := config["collectionKey"]; ok == false {
		return errors.New("collectionKey does not exist in configuration")
	}

	c.collectionKey = c.config["collectionKey"].(string)

	return nil
}

// InDone for Driver interface.
func (c *Collector) Done() error {
	return nil
}

// In for Driver interface.
func (c *Collector) In(query string, args []string, record Record) error {
	// in the future query can be used to specify a different storage key and type
	CollectorStore[c.collectionKey] = append(CollectorStore[c.collectionKey], ResultCollectionItem{
		Record: &record,
		Args:   &args,
	})

	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
func (c *Collector) ExpectedOut() (bool, int, error) {
	return true, len(CollectorStore[c.collectionKey]), nil
}

// Out for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (c *Collector) Out(query string, args []string) (<-chan Record, error) {
	// call Configure with a driver.Config first

	recordChan := make(chan Record, 1)

	go func() {
		if collection, ok := CollectorStore[c.collectionKey]; ok {
			for _, collectionItem := range collection {
				recordChan <- *collectionItem.Record
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
