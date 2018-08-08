package migrate

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"text/template"

	"time"

	"strings"

	"os"

	"net/http"

	"github.com/Masterminds/sprig"
	"github.com/boltdb/bolt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mcuadros/go-candyjs"
	"github.com/satori/go.uuid"
	"github.com/txn2/dmk/cfg"
	"github.com/txn2/dmk/driver"
	"github.com/txn2/dmk/tunnel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// A Runner runs a Migration consisting of a
//  - source DB,
//  - source query,
//  - destination DB
//  - destination query.
//  - a collection of transformers
//
// 1) A query is executed on the on source db and foreach result a transformers is invoked along with the
// collection of transformers.
//
// 2) Each source result becomes transformed data (map[string]interface) and is then used as arguments along with
// a query against the destination database.

// Runner runs migrations.
type RunnerCfg struct {
	Project       Project
	DriverManager *driver.Manager
	TunnelManager tunnel.Manager
	Quiet         bool // Fast mode (no file log / sampled status)
	DryRun        bool
	Verbose       bool
	NoTime        bool   // Disable timestamps and duration for deterministic output
	Limit         int    // Limit the number of records to process
	Path          string // relative path to config
	Logger        *zap.Logger
}

// see NewRunner
type runner struct {
	Cfg     RunnerCfg
	Log     *zap.Logger
	drivers map[string]driver.Driver // store configured drivers
}

var localDbs map[string]*bolt.DB // local bold databases for value mapping

// NewRunner creates and configures a new runner
func NewRunner(cfg RunnerCfg) *runner {

	logger := cfg.Logger

	if logger == nil {
		atom := zap.NewAtomicLevel()
		encoderCfg := zap.NewProductionEncoderConfig()

		if cfg.NoTime {
			// disable timestamps for deterministic output.
			encoderCfg.TimeKey = ""
		}
		logger = zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		))

		atom.SetLevel(zap.DebugLevel)
		defer logger.Sync()
	}

	rnr := &runner{
		Cfg: cfg,
		Log: logger,
	}

	return rnr
}

// getLocalDb gets the database
func (r *runner) getLocalDb(migration string) (*bolt.DB, error) {
	// one database per migration (to avoid dealing with multiple writers)
	dbFile := r.Cfg.Path + r.Cfg.Project.Component.MachineName + "-" + migration + ".db"

	if db, ok := localDbs[dbFile]; ok {
		return db, nil
	}

	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	if localDbs == nil {
		localDbs = make(map[string]*bolt.DB, 1)
	}
	localDbs[dbFile] = db

	return db, nil
}

// configureDriver configures a driver for the migration and database. The configured
// drivers is stored in the event it needs to be re-used in a sub migration.
func (r *runner) configureDriver(migration string, db cfg.Database) (driver.Driver, error) {
	key := migration + "_" + db.Component.MachineName

	if r.drivers == nil {
		r.drivers = make(map[string]driver.Driver, 0)
	}

	// do we have a configured driver for this migration and database?
	if d, ok := r.drivers[key]; ok {
		d.Init()
		return d, nil
	}

	// get a driver of the specified type
	d, err := r.Cfg.DriverManager.GetNewDriver(db.Driver)
	if err != nil {
		return nil, err
	}

	// configure the driver
	err = d.Configure(db.Configuration)
	if err != nil {
		return nil, err
	}

	// store the driver
	r.drivers[key] = d

	d.Init()
	return d, nil
}

// tunnel if needed
func (r *runner) tunnel(database cfg.Database) error {
	// setup a tunnel if needed
	if database.Tunnel != "" {
		if tunnelCfg, ok := r.Cfg.Project.Tunnels[database.Tunnel]; ok {
			err := r.Cfg.TunnelManager.Tunnel(tunnelCfg)
			if err != nil {
				return err
			}

			// take a breath while our tunnel connects
			// TODO: detect connection
			time.Sleep(2 * time.Second)
		}

	}

	return nil
}

// RunResult is returned by the Run method
type RunResult struct {
	MachineName       string
	SourceArgs        []string
	DestinationDriver *driver.Driver
	SourceDriver      *driver.Driver
	Started           time.Time
	Count             int
	Duration          time.Duration
}

// Run runs a migration
func (r *runner) Run(machineName string, sourceArgs []string) (*RunResult, error) {
	migrationStart := time.Now()

	runResult := &RunResult{
		MachineName: machineName,
		SourceArgs:  sourceArgs,
		Started:     migrationStart,
	}

	r.Log.Info("Running Migration",
		zap.String("Type", "Setup"),
		zap.String("MachineName", machineName),
	)

	// get the migration
	migration, ok := r.Cfg.Project.Migrations[machineName]
	if ok != true {
		r.Log.Error("migration: no migration found.",
			zap.String("Type", "Setup"), zap.String("MachineName", machineName))
		return runResult, errors.New("no migration found for " + machineName)
	}

	// get the source db
	sourceDb, ok := r.Cfg.Project.Databases[migration.SourceDb]
	if ok != true {
		r.Log.Error("sourceDb: no source database found. ",
			zap.String("Type", "Setup"), zap.String("MachineName", migration.SourceDb))
		return runResult, errors.New("no source database found for " + migration.SourceDb)
	}

	err := r.tunnel(sourceDb)
	if err != nil {
		r.Log.Error("TunnelError", zap.String("Type", "Setup"), zap.Error(err))
		return runResult, errors.New("unable to tunnel for " + sourceDb.Component.Name)
	}

	// get a driver for the source of migration
	sourceDriver, err := r.configureDriver(machineName, sourceDb)
	if err != nil {
		r.Log.Error("sourceDriver",
			zap.String("Type", "Setup"), zap.Error(err))
		return runResult, err
	}

	// set a pointer to the source driver in the run result
	runResult.SourceDriver = &sourceDriver

	r.Log.Info("Source query args expected.",
		zap.String("Type", "Setup"),
		zap.String("MachineName", machineName),
		zap.Int("ExpectedNArgs", migration.SourceQueryNArgs),
		zap.Int("ReceivedNArgs", len(sourceArgs)),
	)

	if migration.SourceQueryNArgs != len(sourceArgs) {
		r.Log.Error("Unexpected number or arguments received",
			zap.String("Type", "Setup"),
			zap.Int("ExpectedArgs", migration.SourceQueryNArgs),
			zap.Int("ReceivedArgs", len(sourceArgs)),
		)
		return runResult, fmt.Errorf("expecting %d args and got %d", migration.SourceQueryNArgs, len(sourceArgs))
	}

	r.Log.Info("Source query.",
		zap.String("Type", "Setup"),
		zap.String("SourceQuery", strings.Trim(migration.SourceQuery, "\n")),
		zap.Strings("SourceArgs", sourceArgs),
	)

	sourceRecordChan, err := sourceDriver.Out(migration.SourceQuery, sourceArgs)
	if err != nil {
		r.Log.Error("sourceDriver.Out",
			zap.String("Type", "Setup"), zap.Error(err))
		return runResult, err
	}

	r.Log.Info("Migration DestinationDb",
		zap.String("Type", "Setup"),
		zap.String("MachineName", migration.DestinationDb),
	)

	destinationDb, ok := r.Cfg.Project.Databases[migration.DestinationDb]
	if ok != true {
		r.Log.Error("destinationDb: no destination database found.",
			zap.String("Type", "Setup"), zap.String("MachineName", migration.DestinationDb))
		return runResult, errors.New("no destination database found for " + migration.DestinationDb)
	}

	r.Log.Info("Migration Driver",
		zap.String("Type", "Setup"),
		zap.String("MachineName", destinationDb.Driver))

	destinationDriver, err := r.configureDriver(machineName, destinationDb)
	if err != nil {
		return runResult, err
	}
	runResult.DestinationDriver = &destinationDriver

	// javascript transformation script
	script := migration.TransformationScript

	// Javascript engine,
	// see http://duktape.org/ and https://github.com/olebedev/go-duktape
	// see https://github.com/mcuadros/go-candyjs
	ctx := candyjs.NewContext()
	defer ctx.DestroyHeap()
	r.addScriptFunctions(*ctx, machineName)

	queryTemplate, err := template.New("query").Funcs(sprig.TxtFuncMap()).Parse(migration.DestinationQuery)
	if err != nil {
		panic(err)
	}

	setupDuration := time.Now().Sub(migrationStart)
	if r.Cfg.NoTime {
		setupDuration = 0
	}

	r.Log.Info("Start migrating data.",
		zap.String("Type", "Setup"),
		zap.String("FromDb", migration.SourceDb),
		zap.String("ToDb", migration.DestinationDb),
		zap.Duration("SetupDuration", setupDuration),
	)

	count := 0

	// iterate over the sourceRecordChan for driver.Record objects
	for record := range sourceRecordChan {
		count += 1
		recordStart := time.Now()

		args := make([]string, 0)

		// modify r, driver.Record
		if script != "" {
			skipRecord := false
			endMigration := false

			ctx.PushGlobalGoFunction("httpJsonPost", r.HttpJsonPost)

			ctx.PushGlobalGoFunction("getRecord", func() driver.Record {
				return record
			})

			ctx.PushGlobalGoFunction("sendRecord", func(r driver.Record) {
				record = r
			})

			ctx.PushGlobalGoFunction("sendArgs", func(a []string) {
				args = a
			})

			ctx.PushGlobalGoFunction("skip", func() {
				skipRecord = true
			})

			ctx.PushGlobalGoFunction("end", func() {
				endMigration = true
			})

			ctx.EvalString(script)

			// If the transformation script wants us to skip this record
			if skipRecord {
				continue
			}

			// If the transformation script wants us to skip this record
			if endMigration {
				return runResult, nil
			}
		}

		if r.Cfg.DryRun {
			continue
		}

		var query bytes.Buffer
		err = queryTemplate.Execute(&query, record)
		if err != nil {
			return runResult, err
		}

		recDuration := time.Now().Sub(recordStart)
		if r.Cfg.NoTime {
			recDuration = 0
		}

		err = destinationDriver.In(query.String(), args, record)
		if err != nil {
			r.Log.Error("MigrationError",
				zap.Error(err),
				zap.Int("Count", count),
				zap.String("MachineName", machineName),
				zap.String("Query", strings.Trim(query.String(), "\n")),
				zap.Strings("Args", args),
				zap.String("MachineName", machineName),
				zap.Duration("Duration", recDuration),
			)
			return runResult, err
		}

		r.Log.Debug("Status",
			zap.String("Type", "MigrationStatus"),
			zap.Int("Count", count),
			zap.String("MachineName", machineName),
			zap.String("Query", strings.Trim(query.String(), "\n")),
			zap.Strings("Args", args),
			zap.String("MachineName", machineName),
			zap.Duration("Duration", recDuration),
		)

		if r.Cfg.Limit > 0 && r.Cfg.Limit <= count {
			r.Log.Debug("Stopping at specified limit.",
				zap.String("Type", "Done"),
				zap.Int("Count", count),
				zap.String("MachineName", machineName),
				zap.String("Query", strings.Trim(query.String(), "\n")),
				zap.Strings("Args", args),
				zap.String("MachineName", machineName),
				zap.Duration("Duration", recDuration),
			)
			break
		}

	}

	destinationDriver.Done()

	t := time.Now()
	elapsed := t.Sub(migrationStart)
	processingDuration := elapsed - setupDuration

	if r.Cfg.NoTime {
		setupDuration, elapsed, processingDuration = 0, 0, 0
	}

	r.Log.Info("Done with migration.",
		zap.String("MachineName", migration.Component.MachineName),
		zap.String("Type", "Done"),
		zap.Duration("SetupDuration", setupDuration),
		zap.Duration("ProcessingDuration", processingDuration),
		zap.Duration("TotalDuration", elapsed),
		zap.Int("TotalProcessed", count),
	)

	runResult.Duration = elapsed

	return runResult, nil
}

func (r *runner) HttpJsonPost(url, json string) {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Timeout:   time.Second * 60,
		Transport: netTransport,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(json)))
	if err != nil {
		r.Log.Error("HttpNewRequestError", zap.Error(err))
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		r.Log.Error("HttpNewRequestError", zap.Error(err))
	}
	defer resp.Body.Close()

	r.Log.Debug("HttpJsonPostStatus",
		zap.String("Type", "HttpJsonPostStatus"),
		zap.Int("StatusCode", resp.StatusCode),
		zap.String("Status", resp.Status),
	)

}

// addScriptFunctions add utility functions to script context
func (r *runner) addScriptFunctions(ctx candyjs.Context, machineName string) {

	// memory storage
	storage := make(map[string]interface{})

	// recursive migration (sub query) mainly for used with
	// migrations that migrate to a collector
	ctx.PushGlobalGoFunction("run", r.scriptRunner)

	// persistent storage for value maps
	ctx.PushGlobalGoFunction("persistVal", r.persistVal)

	ctx.PushGlobalGoFunction("getMigration", func() string {
		return machineName
	})

	ctx.PushGlobalGoFunction("getUuid", func() string {
		return uuid.NewV4().String()
	})

	ctx.PushGlobalGoFunction("getStorage", func() *map[string]interface{} {
		return &storage
	})

	ctx.PushGlobalGoFunction("sendStorage", func(s map[string]interface{}) {
		storage = s
	})

	ctx.PushGlobalGoFunction("dump", func(obj interface{}) {
		sd := spew.Sdump(obj)
		r.Log.Debug("Script object dump.",
			zap.String("Type", "ScriptOutput"),
			zap.String("MachineName", machineName),
			zap.String("Dump", sd),
		)
	})

	ctx.PushGlobalGoFunction("print", func(obj interface{}) {
		r.Log.Debug("Script output.",
			zap.String("Type", "ScriptOutput"),
			zap.String("MachineName", machineName),
			zap.Any("ScriptPrint", obj),
		)
	})
}

// persistVal gets or stores a fallback value
func (r *runner) persistVal(migration string, k string, fallback string) string {
	db, err := r.getLocalDb(migration)
	if err != nil {
		fmt.Printf("LOCAL DB ERROR: %s", err.Error())
		return ""
	}
	var retVal []byte

	bucket := "persistVal"

	found := false

	// ensure that we have a bucket
	ensureBucket(db, bucket)

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v := b.Get([]byte(k))

		if v != nil {
			retVal = make([]byte, len(v))
			copy(retVal, v)
			found = true
			return nil
		}

		return nil
	})

	if found == true {
		return string(retVal)
	}

	go func() {
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucket))
			err := b.Put([]byte(k), []byte(fallback))
			return err
		})
	}()

	return fallback
}

// scriptRunner returns run function for script context
func (r *runner) scriptRunner(machineNameFromScript string, argsFromScript []string) []driver.ResultCollectionItem {
	runResult, err := r.Run(machineNameFromScript, argsFromScript)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	dd := *runResult.DestinationDriver

	if cdd, ok := dd.(*driver.Collector); ok {
		collection := cdd.GetCollection()

		r.Log.Debug("Number of items Argset will receive from collector.",
			zap.Int("TemCount:", len(collection)))

		return collection
	}

	r.Log.Info("WARNING: run() for %s executed from a script but did not output to a collector.",
		zap.String("MachineName", machineNameFromScript))

	return []driver.ResultCollectionItem{}
}

// ensureBucket makes a bucket if one does not exist
func ensureBucket(db *bolt.DB, bucket string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return err
}
