package driver

import (
	"fmt"
	"strings"

	"errors"

	"log"

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

// HasOutQuery is true for Cassandra
func (c *Cassandra) HasOutQuery() bool {
	return true
}

// HasInQuery is true for Cassandra
func (c *Cassandra) HasInQuery() bool {
	return true
}

// HasCountQuery is true for Cassandra
func (c *Cassandra) HasCountQuery() bool {
	return true
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
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	cluster.Compressor = &gocql.SnappyCompressor{}
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{NumRetries: 3}
	cluster.Consistency = gocql.LocalQuorum

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

	// @TODO implement number of connections in survey
	cluster.NumConns = 1
	session, err := cluster.CreateSession()
	if err != nil {
		panic(err)
	}

	c.session = session

	return nil
}

// InDone for Driver interface.
func (c *Cassandra) Done() error {
	return nil
}

// Done for Driver interface. @TODO implementation
func (c *Cassandra) In(query string, args []string, record Record) error {

	casArgs := make([]interface{}, len(args))
	for i, v := range args {
		casArgs[i] = v
	}

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
	q := c.session.Query(query, casArgs...)
	err := q.Exec()
	if err != nil {
		return err
	}
	q.Release()

	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
// false value mean indefinite.
func (c *Cassandra) ExpectedOut() (bool, int, error) {
	return false, 0, nil
}

// Out for Driver interface. Data coming out of Cassandra
func (c *Cassandra) Out(query string, args []string) (<-chan Record, error) {

	casArgs := make([]interface{}, len(args))
	for i, v := range args {
		casArgs[i] = v
	}

	///casArgs := make([]interface{...args})

	recordChan := make(chan Record, 1)
	q := c.session.Query(query, casArgs...).Consistency(gocql.Quorum)
	fmt.Printf("Out Query: %s\n", q.String())

	itr := q.Iter()

	// check for args
	go func() {
		defer itr.Close()
		defer close(recordChan)

		for {
			// New map each iteration
			row := make(map[string]interface{})
			if !itr.MapScan(row) {
				break
			}
			recordChan <- row
		}
		if err := itr.Close(); err != nil {
			log.Fatal(err)
		}

	}()

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (c *Cassandra) ConfigSurvey(config Config, machineName string) error {
	fmt.Println("---- Cassandra Driver Configuration ----")

	clusterList := ""
	defClusterList := ""
	if v, ok := config["clusterList"]; ok {
		defClusterList = v.(string)
	}
	prompt := &survey.Input{
		Message: "Nodes:",
		Help:    "Comma separated list of nodes ex: \"n1.example.com,n2.example.com\"",
		Default: defClusterList,
	}
	survey.AskOne(prompt, &clusterList, nil)
	config["clusterList"] = clusterList

	keySpace := ""
	defKeySpace := ""
	if v, ok := config["keyspace"]; ok {
		defKeySpace = v.(string)
	}
	prompt = &survey.Input{
		Message: "Keyspace:",
		Help:    "The Cassandra keyspace to query against.",
		Default: defKeySpace,
	}
	survey.AskOne(prompt, &keySpace, nil)
	config["keyspace"] = keySpace

	consistencyNames := make([]string, 0)
	for _, v := range ConsistencyLookup {
		consistencyNames = append(consistencyNames, v)
	}

	consistency := ""
	defConsistency := ""
	if v, ok := config["consistency"]; ok {
		defConsistency = v.(string)
	}
	promptSelect := &survey.Select{
		Message: "Choose a Consistency Level:",
		Options: consistencyNames,
		Default: defConsistency,
	}
	survey.AskOne(promptSelect, &consistency, nil)
	config["consistency"] = consistency

	credentials := false
	defCredentials := false
	if _, ok := config["credentials"]; ok {
		defCredentials = true
	}
	promptBool := &survey.Confirm{
		Message: "Does this cluster require login credentials?",
		Default: defCredentials,
	}
	survey.AskOne(promptBool, &credentials, nil)

	if credentials == true {
		credentialConfig := Config{}
		defCredentialConfig := make(map[interface{}]interface{})

		if _, ok := config["credentials"]; ok {
			defCredentialConfig = config["credentials"].(map[interface{}]interface{})
		}

		username := ""
		defUsername := ""
		if v, ok := defCredentialConfig["username"]; ok {
			defUsername = v.(string)
		}
		prompt = &survey.Input{
			Message: "Username:",
			Default: defUsername,
		}
		survey.AskOne(prompt, &username, nil)
		credentialConfig["username"] = username

		password := ""
		defPassword := ""
		if v, ok := defCredentialConfig["password"]; ok {
			defPassword = v.(string)
		}
		prompt = &survey.Input{
			Message: "Password:",
			Default: defPassword,
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
