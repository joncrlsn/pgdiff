//
// Copyright (c) 2014 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import "fmt"
import "sort"
import "database/sql"
import "github.com/joncrlsn/pgutil"
import "github.com/joncrlsn/misc"

// ==================================
// TableRows definition
// ==================================

// TableRows is a sortable slice of string maps
type TableRows []map[string]string

func (slice TableRows) Len() int {
	return len(slice)
}

func (slice TableRows) Less(i, j int) bool {
	return slice[i]["table_name"] < slice[j]["table_name"]
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

	val := misc.CompareStrings(c.get("table_name"), c2.get("table_name"))
	//fmt.Printf("-- Compared %v: %s with %s \n", val, c.get("table_name"), c2.get("table_name"))
	return val
}

// Add returns SQL to add the table or view
func (c TableSchema) Add() {
	fmt.Printf("CREATE %s %s();", c.get("table_type"), c.get("table_name"))
	fmt.Println()
}

// Drop returns SQL to drop the table or view
func (c TableSchema) Drop() {
	fmt.Printf("DROP %s IF EXISTS %s;\n", c.get("table_type"), c.get("table_name"))
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
	sql := `
SELECT table_schema || '.' || table_name AS table_name
    , CASE table_type WHEN 'BASE TABLE' THEN 'TABLE' ELSE table_type END AS table_type
    , is_insertable_into
FROM information_schema.tables 
WHERE table_schema NOT LIKE 'pg_%' 
WHERE table_schema <> 'information_schema' 
AND table_type = 'BASE TABLE'
ORDER BY table_name;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

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
