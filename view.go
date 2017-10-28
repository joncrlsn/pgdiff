//
// Copyright (c) 2016 Jon Carlson.  All rights reserved.
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
// ViewRows definition
// ==================================

// ViewRows is a sortable slice of string maps
type ViewRows []map[string]string

func (slice ViewRows) Len() int {
	return len(slice)
}

func (slice ViewRows) Less(i, j int) bool {
	return slice[i]["viewname"] < slice[j]["viewname"]
}

func (slice ViewRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ViewSchema holds a channel streaming view information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// ViewSchema implements the Schema interface defined in pgdiff.go
type ViewSchema struct {
	rows   ViewRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *ViewSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *ViewSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *ViewSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ViewSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a ViewSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("viewname"), c2.get("viewname"))
	//fmt.Printf("-- Compared %v: %s with %s \n", val, c.get("viewname"), c2.get("viewname"))
	return val
}

// Add returns SQL to create the view
func (c ViewSchema) Add() {
	fmt.Printf("CREATE VIEW %s AS %s \n\n", c.get("viewname"), c.get("definition"))
}

// Drop returns SQL to drop the view
func (c ViewSchema) Drop() {
	fmt.Printf("DROP VIEW %s;\n\n", c.get("viewname"))
}

// Change handles the case where the names match, but the definition does not
func (c ViewSchema) Change(obj interface{}) {
	c2, ok := obj.(*ViewSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a ViewSchema instance", c2)
	}
	if c.get("definition") != c2.get("definition") {
		fmt.Printf("DROP VIEW %s;\n", c.get("viewname"))
		fmt.Printf("CREATE VIEW %s AS %s \n\n", c.get("viewname"), c.get("definition"))
	}
}

// compareViews outputs SQL to make the views match between DBs
func compareViews(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
	SELECT schemaname || '.' || viewname AS viewname
		, definition 
	FROM pg_views 
	WHERE schemaname NOT LIKE 'pg_%' 
	ORDER BY viewname;
	`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(ViewRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(ViewRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here
	var schema1 Schema = &ViewSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &ViewSchema{rows: rows2, rowNum: -1}

	// Compare the views
	doDiff(schema1, schema2)
}
