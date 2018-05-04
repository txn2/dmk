package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binaryName = "dmk"
var update = flag.Bool("update", false, "update .golden files")

// TestMain sets up a binary for functional tests.
func TestMain(m *testing.M) {
	make := exec.Command("make")
	err := make.Run()
	if err != nil {
		fmt.Printf("could not make binary for %s: %v", binaryName, err)
		os.Exit(1)
	}

	// run tests
	res := m.Run()

	makeClean := exec.Command("make", "clean")
	err = makeClean.Run()
	if err != nil {
		fmt.Printf("could not remove binary for %s: %v", binaryName, err)
		os.Exit(1)
	}

	os.Exit(res)
}

// TestCliGoldenArgs tests deterministic output data.
func TestCliGoldenArgs(t *testing.T) {

	tests := []struct {
		name    string
		args    []string
		fixture string
	}{
		{"help",
			[]string{"help"},
			"help.golden",
		},
		{"example_csv_to_cassandra",
			[]string{
				"-d", "examples",
				"-p", "example",
				"run",
				"-v", // verbose
				"-n", // disable timestamps for deterministic output.
				"-l", // log out (log to standard out)
				"example_csv_to_cassandra", // migration
			}, "example_csv_to_cassandra.golden",
		},
		{"cassandra_to_cassandra_by_name",
			[]string{
				"-d", "examples",
				"-p", "example",
				"run",
				"-v", // verbose
				"-n", // disable timestamps for deterministic output.
				"-l", // log out (log to standard out)
				"cassandra_to_cassandra_by_name", // migration
				"example",                        // expected argument
			}, "cassandra_to_cassandra_by_name.golden",
		},
		{"cassandra_to_cassandra_name_lookup",
			[]string{
				"-d", "examples",
				"-p", "example",
				"run",
				"-v", // verbose
				"-n", // disable timestamps for deterministic output.
				"-l", // log out (log to standard out)
				"cassandra_to_cassandra_name_lookup", // migration
				"example", // expected argument
			}, "cassandra_to_cassandra_name_lookup.golden",
		},
		{"cassandra_to_cassandra_using_collector",
			[]string{
				"-d", "examples",
				"-p", "example",
				"run",
				"-v", // verbose
				"-n", // disable timestamps for deterministic output.
				"-l", // log out (log to standard out)
				"cassandra_to_cassandra_using_collector", // migration
				"example", // expected argument
			}, "cassandra_to_cassandra_using_collector.golden",
		},
	}

	for _, tt := range tests {

		c := exec.Command("./"+binaryName, tt.args...)
		output, err := c.CombinedOutput()
		if err != nil {
			log.Printf(err.Error())
			log.Printf("%s", output)
			t.Fatal(err)
		}

		gp := filepath.Join("testdata", tt.fixture)
		if *update {
			t.Log("update golden file")
			if err := ioutil.WriteFile(gp, output, 0644); err != nil {
				t.Fatalf("failed to update golden file: %s", err)
			}
		}

		g, err := ioutil.ReadFile(gp)
		if err != nil {
			t.Fatalf("failed reading .golden: %s", err)
		}
		t.Log(string(output))
		if !bytes.Equal(output, g) {
			t.Errorf("output does not match .golden file")
		}

	}
}
