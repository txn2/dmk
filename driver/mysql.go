package driver

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey"
	"github.com/davecgh/go-spew/spew"
	_ "github.com/go-sql-driver/mysql"
)

// MySql implements data.Driver
type MySql struct {
	config Config
	db     *sql.DB
}

// Configure (keys determined in ConfigSurvey)
func (m *MySql) Configure(config Config) error {
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
		port = "33306"
		//return errors.New("missing config key databasePort")
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

	database, err := sql.Open("mysql", username+":"+password+"@tcp("+host+":"+port+")/"+dbName)

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
func (m *MySql) In(query string) error {
	fmt.Printf("MySql In is not yet implemented.\n")
	return nil
}

// Out for Driver interface.
func (m *MySql) Out(query string, args Args) (<-chan Record, error) {
	// call Configure with a driver.Config first
	if m.db == nil {
		return nil, errors.New("MySql is not configured")
	}

	recordChan := make(chan Record, 1)

	database := m.db

	rows, err := database.Query(query)

	if err != nil {
		println("ROWS ERROR: " + err.Error())
		return nil, errors.New(err.Error())
	}

	cols, _ := rows.Columns()

	if err != nil {
		println("COLS ERROR: " + err.Error())
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	println("HERE I AM")

	for rows.Next() {
		println("IN ROWS NEXT")
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			errors.New(err.Error())
		}

		record := Record{}
		for i, colName := range cols {
			val := columnPointers[i].(interface{})
			vv := val.(interface{})
			record[colName] = *vv.(*interface{})

			spew.Dump(*columnPointers[i].(*interface{}))
		}
		s := "this is a test for myke"
		val := make(map[string]interface{})
		val["test"] = &s
		//fmt.Printf("%s", val)

		vv := val["test"].(interface{})

		spew.Dump(*vv.(*string))
		// send the record out the channel
		recordChan <- record
	}
	err = rows.Err()
	if err != nil {
		errors.New(err.Error())
	}

	return recordChan, nil
}

// ConfigSurvey is an implementation of Driver
func (m *MySql) ConfigSurvey(config Config) error {
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
