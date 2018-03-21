package driver

import (
	"fmt"
	"strings"

	"errors"

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
	// @TODO improve validation
	fmt.Printf("Configuring Cassandra\n")

	// get cluster nodes
	nodes := strings.Split(config["clusterList"].(string), ",")

	// Create a database session
	// see https://github.com/scylladb/gocqlx
	cluster := gocql.NewCluster(nodes...)

	if credentialsInt, ok := config["credentials"].(map[interface{}]interface{}); ok {
		// create u/p slice
		up := make(map[string]string)
		for i, v := range credentialsInt {
			up[i.(string)] = v.(string)
		}

		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: up["username"],
			Password: up["password"],
		}
	}

	cluster.Keyspace = config["keyspace"].(string)

	// @TODO implement consistency lookup (for not default to Quorum)
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()
	if err != nil {
		panic(err)
	}

	c.session = session

	return nil
}

// Done for Driver interface.
func (c *Cassandra) Done() error {
	return nil
}

// In for Driver interface. @TODO implementation
func (c *Cassandra) In(query string) error {

	if c.session == nil {
		return errors.New("the Cassandra driver is not configured")
	}

	// @TODO: implement batching
	// make a new batch and store in the Type
	// keep a counter and execute the batch on interval
	// execute remaining batch on finish

	// execute the query
	// see https://gocql.github.io/
	// see https://godoc.org/github.com/gocql/gocql
	q := c.session.Query(query)
	err := q.Exec()
	if err != nil {
		return err
	}

	return nil
}

// Out for Driver interface. CSV ignores the query and args, reading
// the entire file and streaming each record as lines are parsed.
func (c *Cassandra) Out(query string, args Args) (<-chan Record, error) {
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
