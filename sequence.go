//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
	"sort"
	"text/template"
)

var (
	sequenceSqlTemplate = initSequenceSqlTemplate()
)

// Initializes the Sql template
func initSequenceSqlTemplate() *template.Template {
	sql := `
SELECT sequence_schema AS schema_name
    , {{if eq $.DbSchema "*" }}sequence_schema || '.' || {{end}}sequence_name AS compare_name
    , sequence_name 
	, data_type
	, start_value
	, minimum_value
	, maximum_value
	, increment
	, cycle_option 
FROM information_schema.sequences
WHERE true
{{if eq $.DbSchema "*" }}
AND sequence_schema NOT LIKE 'pg_%' 
AND sequence_schema <> 'information_schema' 
{{else}}
AND sequence_schema = '{{$.DbSchema}}'
{{end}}
`

	t := template.New("SequenceSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// SequenceRows definition
// ==================================

// SequenceRows is a sortable slice of string maps
type SequenceRows []map[string]string

func (slice SequenceRows) Len() int {
	return len(slice)
}

func (slice SequenceRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}

func (slice SequenceRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// SequenceSchema holds a channel streaming sequence information from one of the databases as well as
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

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	return val
}

// Add returns SQL to add the sequence
func (c SequenceSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}
	fmt.Printf("CREATE SEQUENCE %s.%s INCREMENT %s MINVALUE %s MAXVALUE %s START %s;\n", schema, c.get("sequence_name"), c.get("increment"), c.get("minimum_value"), c.get("maximum_value"), c.get("start_value"))
}

// Drop returns SQL to drop the sequence
func (c SequenceSchema) Drop() {
	fmt.Printf("DROP SEQUENCE %s.%s;\n", c.get("schema_name"), c.get("sequence_name"))
}

// Change doesn't do anything right now.
func (c SequenceSchema) Change(obj interface{}) {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, Change(obj) needs a SequenceSchema instance", c2)
	}
	// Don't know of anything helpful we should do here
}

// compareSequences outputs SQL to make the sequences match between DBs or schemas
func compareSequences(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	sequenceSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	sequenceSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

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

	// We have to explicitly type this as Schema here for some unknown (to me) reason
	var schema1 Schema = &SequenceSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &SequenceSchema{rows: rows2, rowNum: -1}

	// Compare the sequences
	doDiff(schema1, schema2)
}
