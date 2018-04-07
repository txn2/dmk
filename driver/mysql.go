package driver

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey"
	_ "github.com/go-sql-driver/mysql" // driver import
)

// MySql implements data.Driver
type MySql struct {
	config Config
	db     *sql.DB
}

// HasOutQuery is true for MySql
func (m *MySql) HasOutQuery() bool {
	return true
}

// HasInQuery is true for MySql
func (m *MySql) HasInQuery() bool {
	return true
}

// HasCountQuery is true for MySql
func (m *MySql) HasCountQuery() bool {
	return true
}

// Configure (keys determined in ConfigSurvey)
func (m *MySql) Configure(config Config) error {
	fmt.Printf("Configuring a MySQL driver.\n")

	// Validation
	dbName, ok := config["databaseName"].(string)
	if ok != true {
		return errors.New("missing config key databaseName")
	}

	host, ok := config["databaseHost"].(string)
	if ok != true {
		return errors.New("missing config key databaseHost")
	}

	port, ok := config["databasePort"].(string)
	if ok != true {
		return errors.New("missing config key databasePort")
	}

	username, ok := config["username"].(string)
	if ok != true {
		return errors.New("missing credential key username")
	}

	password := ""

	if credentialsInt, ok := config["credentials"].(map[interface{}]interface{}); ok {
		if ok != true {
			return errors.New("configured value of MySql credentials is not an interface")
		}
		// create u/p slice
		up := make(map[string]string)
		for i, v := range credentialsInt {
			up[i.(string)] = v.(string)
		}
		password, ok = up["password"]
		if ok != true {
			return errors.New("missing credential key password")
		}
	}

	connectionStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, dbName)
	fmt.Printf("MySql driver connecting to: %s\n", connectionStr)

	database, err := sql.Open("mysql", connectionStr)
	if err != nil {
		return errors.New(err.Error())
	}

	//defer database.Close()

	err = database.Ping()
	if err != nil {
		return errors.New(err.Error())
	}
	m.db = database

	m.config = config

	return nil
}

// Done for Driver interface.
func (m *MySql) Done() error {
	return nil
}

// In for Driver interface. @TODO implementation
func (m *MySql) In(query string, args []string, record Record) error {
	fmt.Printf("MySql In is not yet implemented.\n")
	return nil
}

// ExpectedOut returns true and the number of expected outbound records,
// false value mean indefinite.
// TODO: implement expected out for MySQL
func (m *MySql) ExpectedOut() (bool, int, error) {
	return false, 0, nil
}

// Out for Driver interface.
func (m *MySql) Out(query string, args []string) (<-chan Record, error) {
	// call Configure with a driver.Config first
	if m.db == nil {
		return nil, errors.New("MySql is not configured")
	}

	recordChan := make(chan Record, 1)

	database := m.db

	myArgs := make([]interface{}, len(args))
	for i, v := range args {
		myArgs[i] = v
	}

	rows, err := database.Query(query, myArgs...)
	if err != nil {
		return nil, err
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	go func() {
		defer rows.Close()
		for rows.Next() {

			colsRef := make([]string, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range cols {
				columnPointers[i] = &colsRef[i]
			}

			err = rows.Scan(columnPointers...)
			if err != nil {
				panic(err.Error())
			}

			record := Record{}
			for i, col := range cols {
				cp := columnPointers[i]
				record[col] = *cp.(*string)
			}

			recordChan <- record
		}

		// fell out of loop
		close(recordChan)

	}()

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (m *MySql) ConfigSurvey(config Config, machineName string) error {
	fmt.Println("---- MySql Driver Configuration ----")

	host := ""
	prompt := &survey.Input{
		Message: "Database Host:",
		Help:    "The host of the MySql database.",
	}
	survey.AskOne(prompt, &host, nil)
	config["databaseHost"] = host

	port := ""
	prompt = &survey.Input{
		Message: "Database Port:",
		Default: "3306",
		Help:    "The port of the MySql database.",
	}
	survey.AskOne(prompt, &port, nil)
	config["databasePort"] = port

	username := ""
	prompt = &survey.Input{
		Message: "Username:",
		Help:    "The MySql database username.",
	}
	survey.AskOne(prompt, &username, nil)
	config["username"] = username

	credentials := false
	promptBool := &survey.Confirm{
		Message: "Does this MySql database require login password?",
	}
	survey.AskOne(promptBool, &credentials, nil)

	if credentials == true {
		credentialConfig := Config{}

		password := ""
		prompt = &survey.Input{
			Message: "Password:",
			Help:    "The MySql database password.",
		}
		survey.AskOne(prompt, &password, nil)
		credentialConfig["password"] = password

		config["credentials"] = credentialConfig
	}

	dbName := ""
	prompt = &survey.Input{
		Message: "Database Name:",
		Help:    "The MySql database name to query against.",
	}
	survey.AskOne(prompt, &dbName, nil)
	config["databaseName"] = dbName

	return nil
}

// Register this driver with the driver manager
func init() {
	DriverManager.AddDriver("mysql", func() Driver { return new(MySql) })
}
