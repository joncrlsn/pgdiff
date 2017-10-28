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
	tableSqlTemplate = initTableSqlTemplate()
)

// Initializes the Sql template
func initTableSqlTemplate() *template.Template {

	sql := `
SELECT table_schema
    , {{if eq $.DbSchema "*" }}table_schema || '.' || {{end}}table_name AS compare_name
	, table_name
    , CASE table_type 
	  WHEN 'BASE TABLE' THEN 'TABLE' 
	  ELSE table_type END AS table_type
    , is_insertable_into
FROM information_schema.tables 
WHERE table_type = 'BASE TABLE'
{{if eq $.DbSchema "*" }}
AND table_schema NOT LIKE 'pg_%' 
AND table_schema <> 'information_schema' 
{{else}}
AND table_schema = '{{$.DbSchema}}'
{{end}}
ORDER BY compare_name;
`
	t := template.New("TableSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// TableRows definition
// ==================================

// TableRows is a sortable slice of string maps
type TableRows []map[string]string

func (slice TableRows) Len() int {
	return len(slice)
}

func (slice TableRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}

func (slice TableRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// TableSchema holds a channel streaming table information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// TableSchema implements the Schema interface defined in pgdiff.go
type TableSchema struct {
	rows   TableRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *TableSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *TableSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *TableSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a TableSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	//fmt.Printf("-- Compared %v: %s with %s \n", val, c.get("table_name"), c2.get("table_name"))
	return val
}

// Add returns SQL to add the table or view
func (c TableSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("table_schema")
	}
	fmt.Printf("CREATE %s %s.%s();", c.get("table_type"), schema, c.get("table_name"))
	fmt.Println()
}

// Drop returns SQL to drop the table or view
func (c TableSchema) Drop() {
	fmt.Printf("DROP %s %s.%s;\n", c.get("table_type"), c.get("table_schema"), c.get("table_name"))
}

// Change handles the case where the table and column match, but the details do not
func (c TableSchema) Change(obj interface{}) {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a TableSchema instance", c2)
	}
	// There's nothing we need to do here
}

// compareTables outputs SQL to make the table names match between DBs
func compareTables(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	tableSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	tableSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(TableRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(TableRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here
	var schema1 Schema = &TableSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &TableSchema{rows: rows2, rowNum: -1}

	// Compare the tables
	doDiff(schema1, schema2)
}
