package csvmap

import (
	"bytes"
	"io"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO test csv.Reader.Read() error

func TestReadHeader(t *testing.T) {
	Convey("Given a reader", t, func() {
		Convey("If there is no next row", func() {
			buf := bytes.NewBuffer(nil)
			columns, err := NewReader(buf).ReadHeader()
			Convey("No columns should be returned", func() { So(columns, ShouldBeEmpty) })
			Convey("The error should be io.EOF", func() { So(err, ShouldEqual, io.EOF) })
		})
		Convey("If the number of columns doesn't match", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar"))
			reader.Reader.FieldsPerRecord = 3
			columns, err := reader.ReadHeader()
			Convey("A record should be returned", func() { So(columns, ShouldResemble, []string{"foo", "bar"}) })
			Convey("The error message should contain...", func() { So(err.Error(), ShouldContainSubstring, "wrong number of fields in line") })
		})
		Convey("If there is a next row", func() {
			expectedColumns := []string{"foo", "bar", "baz"}
			buf := bytes.NewBufferString(strings.Join(expectedColumns, ","))
			columns, err := NewReader(buf).ReadHeader()
			Convey("Columns should be returned", func() { So(columns, ShouldResemble, expectedColumns) })
			Convey("No error should be returned", func() { So(err, ShouldBeNil) })
		})
	})
}

func TestRead(t *testing.T) {
	Convey("Given a reader", t, func() {
		Convey("If there is no next row", func() {
			record, err := NewReader(bytes.NewBuffer(nil)).Read()
			Convey("No record should be returned", func() { So(record, ShouldBeEmpty) })
			Convey("The error should be io.EOF", func() { So(err, ShouldEqual, io.EOF) })
		})
		Convey("If there are two indices with the same name", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar"))
			reader.Columns = []string{"column", "column"}
			record, err := reader.Read()
			Convey("No record should be returned", func() { So(record, ShouldBeEmpty) })
			Convey("The error should be...", func() { So(err.Error(), ShouldContainSubstring, "Multiple indices with the same name") })
		})
		Convey("If the number of columns doesn't match and reader.Reader.FieldsPerRecord is set", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar"))
			reader.Columns = []string{"A", "Day", "In", "The", "Life"}
			reader.Reader.FieldsPerRecord = 5
			columns, err := reader.Read()
			Convey("A record should be returned", func() { So(columns, ShouldResemble, map[string]string{"A": "foo", "Day": "bar"}) })
			Convey("The error message should contain...", func() { So(err.Error(), ShouldContainSubstring, "wrong number of fields in line") })
		})
		Convey("If the row contains fewer columns than Reader.Columns", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar"))
			reader.Columns = []string{"Hey", "hey", "what", "can", "I", "do"}
			record, err := reader.Read()
			Convey("A record should be returned", func() { So(record, ShouldResemble, map[string]string{"Hey": "foo", "hey": "bar"}) })
			Convey("No error should be returned", func() { So(err, ShouldBeNil) })
		})
		Convey("If the row contains more columns than Reader.Columns", func() {
			reader := NewReader(bytes.NewBufferString("another,one,bites,the,dust"))
			reader.Columns = []string{"New", "Song"}
			record, err := reader.Read()
			Convey("A record should be returned", func() { So(record, ShouldResemble, map[string]string{"New": "another", "Song": "one"}) })
			Convey("No error should be returned", func() { So(err, ShouldBeNil) })
		})
		Convey("If the row contains the same number of fields as columns int Reader.Columns", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar"))
			reader.Columns = []string{"stuff", "more stuff"}
			record, err := reader.Read()
			Convey("A record should be returned", func() { So(record, ShouldResemble, map[string]string{"stuff": "foo", "more stuff": "bar"}) })
			Convey("No error should be returned", func() { So(err, ShouldBeNil) })
		})
	})
}

func TestReadAll(t *testing.T) {
	Convey("Given a reader", t, func() {
		Convey("If there is no next row", func() {
			records, err := NewReader(bytes.NewBuffer(nil)).ReadAll()
			Convey("Empty records should be returned", func() { So(records, ShouldBeEmpty) })
			Convey("No error should be returned", func() { So(err, ShouldBeNil) })
		})
		Convey("If there is an error reading one of the rows", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar"))
			reader.Reader.FieldsPerRecord = 3
			records, err := reader.ReadAll()
			Convey("No record should be returned", func() { So(records, ShouldBeEmpty) })
			Convey("The error message should contain", func() { So(err.Error(), ShouldContainSubstring, "wrong number of fields in line") })
		})
		Convey("If everything is good", func() {
			reader := NewReader(bytes.NewBufferString("foo,bar\nfiz,buz"))
			reader.Columns = []string{"stuff", "more stuff"}
			records, err := reader.ReadAll()
			Convey("Records record should be returned", func() {
				expected := []map[string]string{
					map[string]string{"stuff": "foo", "more stuff": "bar"},
					map[string]string{"stuff": "fiz", "more stuff": "buz"},
				}
				So(records, ShouldResemble, expected)
			})
			Convey("No error should be returned", func() { So(err, ShouldBeNil) })
		})
	})
}
