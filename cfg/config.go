package cfg

type Component struct {
	Kind        string
	Name        string
	MachineName string `yaml:"machineName"`
	Description string
}

type Credentials struct {
	Component Component
	Username  string
	Password  string
}

type Database struct {
	Component   Component
	Driver      string
	Location    string
	Credentials Credentials
}

type Query struct {
	Component Component
	Statement string
}

type Migration struct {
	Component          Component
	SourceDb           string // db machine name
	DestinationDb      string // db machine name
	SourceQuery        string // how to get the data
	DestinationQuery   string // where to put the data
	TransformationSpec interface{}
}

type Project struct {
	Component  Component
	Databases  map[string]Database  // map of database machine names to databases
	Queries    map[string]Query     // map of query machine names to queries
	Migrations map[string]Migration // map or migration machine names to migrations
}
