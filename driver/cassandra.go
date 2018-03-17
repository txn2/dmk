package driver

import (
	"fmt"

	"github.com/AlecAivazis/survey"
	"github.com/gocql/gocql"
)

// ConsistencyLookup is a map of Cassandra consistency types.
// see https://docs.datastax.com/en/cassandra/3.0/cassandra/dml/dmlClientRequestsReadExp.html?hl=consistency
var ConsistencyLookup = map[gocql.Consistency]string{
	gocql.Any:         `Any`,
	gocql.One:         `One`,
	gocql.LocalOne:    `LocalOne`,
	gocql.LocalQuorum: `LocalQuorum`,
	gocql.Quorum:      `Quorum`,
	gocql.All:         `All`,
}

// Cassandra implements data.Driver
type Cassandra struct {
	session *gocql.Session
	config  Config
}

// Configure (keys determined in ConfigSurvey)
func (c *Cassandra) Configure(config Config) error {
	return nil
}

// Execute for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (c *Cassandra) Execute(query string, args Args) (chan Record, error) {
	fmt.Printf("Cassandra executor is not yet functional\n")

	recordChan := make(chan Record, 1)
	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (c *Cassandra) ConfigSurvey(config Config) error {
	fmt.Println("---- Cassandra Driver Configuration ----")

	clusterList := ""
	prompt := &survey.Input{
		Message: "Nodes:",
		Help:    "Comma separated list of nodes ex: \"n1.example.com,n2.example.com\"",
	}
	survey.AskOne(prompt, &clusterList, nil)
	config["clusterList"] = clusterList

	keyspace := ""
	prompt = &survey.Input{
		Message: "Keyspace:",
		Help:    "The Cassandra keyspace to query against.",
	}
	survey.AskOne(prompt, &keyspace, nil)
	config["keyspace"] = keyspace

	consistencyNames := make([]string, 0)
	for _, v := range ConsistencyLookup {
		consistencyNames = append(consistencyNames, v)
	}

	consistency := ""
	promptSelect := &survey.Select{
		Message: "Choose a Consistency Level:",
		Options: consistencyNames,
	}
	survey.AskOne(promptSelect, &consistency, nil)
	config["consistency"] = consistency

	credentials := false
	promptBool := &survey.Confirm{
		Message: "Does this cluster require login credentials?",
	}
	survey.AskOne(promptBool, &credentials, nil)

	if credentials == true {
		credentialConfig := Config{}

		username := ""
		prompt = &survey.Input{
			Message: "Username:",
		}
		survey.AskOne(prompt, &username, nil)
		credentialConfig["username"] = username

		password := ""
		prompt = &survey.Input{
			Message: "Password:",
		}
		survey.AskOne(prompt, &password, nil)
		credentialConfig["password"] = password

		config["credentials"] = credentialConfig
	}

	// populate
	c.config = config

	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("cassandra", func() Driver { return new(Cassandra) })
}
