//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import "sort"
import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"
import "github.com/joncrlsn/misc"

// ==================================
// SequenceRows definition
// ==================================

// SequenceRows is a sortable slice of string maps
type SequenceRows []map[string]string

func (slice SequenceRows) Len() int {
	return len(slice)
}

func (slice SequenceRows) Less(i, j int) bool {
	return slice[i]["sequence_name"] < slice[j]["sequence_name"]
}

func (slice SequenceRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// SequenceSchema holds a channel streaming table information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// SequenceSchema implements the Schema interface defined in pgdiff.go
type SequenceSchema struct {
	rows   SequenceRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *SequenceSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *SequenceSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *SequenceSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a SequenceSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("sequence_name"), c2.get("sequence_name"))
	return val
}

// Add returns SQL to add the table
func (c SequenceSchema) Add() {
	fmt.Printf("CREATE SEQUENCE %s INCREMENT %s MINVALUE %s MAXVALUE %s START %s;\n", c.get("sequence_name"), c.get("increment"), c.get("minimum_value"), c.get("maximum_value"), c.get("start_value"))

}

// Drop returns SQL to drop the table
func (c SequenceSchema) Drop() {
	fmt.Printf("DROP SEQUENCE IF EXISTS %s;\n", c.get("sequence_name"))
}

// Change handles the case where the table and column match, but the details do not
func (c SequenceSchema) Change(obj interface{}) {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, Change(obj) needs a SequenceSchema instance", c2)
	}
	// Don't know of anything helpful we should do here
}

// compareSequences outputs SQL to make the sequences match between DBs
func compareSequences(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT sequence_schema || '.' || sequence_name
	, data_type
	, start_value
	, minimum_value
	, maximum_value
	, increment
	, cycle_option 
FROM information_schema.sequences
WHERE sequence_schema NOT LIKE 'pg_%'
ORDER BY sequence_name;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(SequenceRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(SequenceRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &SequenceSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &SequenceSchema{rows: rows2, rowNum: -1}

	// Compare the tables
	doDiff(schema1, schema2)
}
