package main

import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"

// SequenceSchema holds a channel streaming table information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// SequenceSchema implements the Schema interface defined in pgdiff.go
type SequenceSchema struct {
	channel chan map[string]string
	row     map[string]string
	done    bool
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *SequenceSchema) NextRow() bool {
	c.row = <-c.channel
	if len(c.row) == 0 {
		c.done = true
	}
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *SequenceSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a SequenceSchema instance", c2)
		return +999
	}

	val := _compareString(c.row["sequence_name"], c2.row["sequence_name"])
	return val
}

// Add returns SQL to add the table
func (c SequenceSchema) Add() {
	fmt.Printf("CREATE SEQUENCE %s INCREMENT %s MINVALUE %s MAXVALUE %s START %s;\n", c.row["sequence_name"], c.row["increment"], c.row["minimum_value"], c.row["maximum_value"], c.row["start_value"])

}

// Drop returns SQL to drop the table
func (c SequenceSchema) Drop() {
	fmt.Printf("DROP SEQUENCE IF EXISTS %s;\n", c.row["sequence_name"])
}

// Change handles the case where the table and column match, but the details do not
func (c SequenceSchema) Change(obj interface{}) {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a SequenceSchema instance", c2)
	}
}

// compareSequences outputs SQL to make the sequences match between DBs
func compareSequences(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT sequence_name, data_type, start_value
	, minimum_value, maximum_value
	, increment, cycle_option 
FROM information_schema.sequences
WHERE sequence_schema = 'public'
ORDER BY sequence_name COLLATE "C" ASC;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &SequenceSchema{channel: rowChan1}
	var schema2 Schema = &SequenceSchema{channel: rowChan2}

	// Compare the tables
	doDiff(schema1, schema2)
}
