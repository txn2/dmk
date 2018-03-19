# go-csv-map
A wrapper around golang's encoding/csv providing map-based access

## Installation

    go get github.com/recursionpharma/go-csv-map

## Usage

### Basic Usage

#### CSV with first line as header

    import (
        "bytes"
        "fmt"
        "os"

        "github.com/recursionpharma/go-csv-map"
    )

    buf := bytes.NewBufferString("Album,Year\nDark Side of the Moon,1973\nExile On Main St,1972")
    reader := csvmap.NewReader(buf)
    reader.Columns, err := reader.ReadHeader()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    records, err := reader.ReadAll()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    fmt.Println(records[1]["Year"])
    // Output: 1972

#### CSV with no header line

    import (
        "bytes"
        "fmt"
        "os"

        "github.com/recursionpharma/go-csv-map"
    )

    buf := bytes.NewBufferString("Dark Side of the Moon,1973\nExile On Main St,1972")
    reader := csvmap.NewReader(buf)
    reader.Columns = []string{"Album", "Year"}
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    records, err := reader.ReadAll()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    fmt.Println(records[1]["Year"])
    // Output: 1972

### Advanced Usage

`csvmap.Reader.Reader` gives you access to the underlying [`csv.Reader`](https://golang.org/pkg/encoding/csv/#Reader) object.

#### Changing the separator

    import (
        "bytes"
        "fmt"
        "os"

        "github.com/recursionpharma/go-csv-map"
    )

    buf := bytes.NewBufferString("Album;Year\nDark Side of the Moon;1973\nExile On Main St;1972")
    reader := csvmap.NewReader(buf)
    reader.Reader.Comma = ';'
    reader.Columns, err := reader.ReadHeader()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    records, err := reader.ReadAll()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    fmt.Println(records[1]["Year"])
    // Output: 1972

#### Checking for the number of fields

    import (
        "bytes"
        "fmt"
        "os"

        "github.com/recursionpharma/go-csv-map"
    )

    buf := bytes.NewBufferString("Album;Year\nDark Side of the Moon;1973\nExile On Main St;1972")
    reader := csvmap.NewReader(buf)
    reader.Reader.FieldsPerRecord = 3 
    reader.Columns, err := reader.ReadHeader()
    fmt.Println(err != nil)
    // Output: true

## Directory Structure

```
code/go-csv-map/
|-- csvmap.go
|   Main code
|-- csvmap_test.go
|   Tests
|-- examples_test.go
|   Examples
|-- .gitignore
|   Files git will ignore
|-- LICENSE
|   MIT License
|-- README.md
|   This file
`-- .travis.yml
    Travis configuration
```

The above file tree was generated with `tree -a -L 1 --charset ascii`.
