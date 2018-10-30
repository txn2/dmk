package csvmap

import (
	"bytes"
	"fmt"
)

func ExampleNewReader() {
	reader := NewReader(bytes.NewBuffer(nil))
	fmt.Println(string(reader.Reader.Comma))
	// Output: ,
}

func ExampleReader_ReadHeader() {
	reader := NewReader(bytes.NewBufferString("first,last\nAlexander,Hamilton"))
	var err error
	reader.Columns, err = reader.ReadHeader()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(reader.Columns[1])
	// Output: last
}

func ExampleReader_Read() {
	reader := NewReader(bytes.NewBufferString("first,last\nAlexander,Hamilton"))
	var err error
	reader.Columns, err = reader.ReadHeader()
	if err != nil {
		fmt.Println(err)
		return
	}
	record, err := reader.Read()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(record["first"])
	// Output: Alexander
}

func ExampleReader_ReadAll() {
	reader := NewReader(bytes.NewBufferString("first,last\nAlexander,Hamilton\nAaron,Burr"))
	var err error
	reader.Columns, err = reader.ReadHeader()
	if err != nil {
		fmt.Println(err)
		return
	}
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(records[1]["last"])
	// Output: Burr
}
