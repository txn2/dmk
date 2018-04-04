package driver

type ArgSet []interface{}

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
