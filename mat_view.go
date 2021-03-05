//
// Copyright (c) 2016 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
)

// ==================================
// MatViewRows definition
// ==================================

// MatViewRows is a sortable slice of string maps
type MatViewRows []map[string]string

func (slice MatViewRows) Len() int {
	return len(slice)
}

func (slice MatViewRows) Less(i, j int) bool {
	return slice[i]["matviewname"] < slice[j]["matviewname"]
}

func (slice MatViewRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// MatViewSchema holds a channel streaming matview information from one of the databases as well as
// a reference to the current row of data we're matviewing.
//
// MatViewSchema implements the Schema interface defined in pgdiff.go
type MatViewSchema struct {
	rows   MatViewRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *MatViewSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *MatViewSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *MatViewSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*MatViewSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a MatViewSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("matviewname"), c2.get("matviewname"))
	//fmt.Printf("-- Compared %v: %s with %s \n", val, c.get("matviewname"), c2.get("matviewname"))
	return val
}

// Add returns SQL to create the matview
func (c MatViewSchema) Add() {
	fmt.Printf("CREATE MATERIALIZED VIEW %s AS %s \n\n%s \n\n", c.get("matviewname"), c.get("definition"), c.get("indexdef"))
}

// Drop returns SQL to drop the matview
func (c MatViewSchema) Drop() {
	fmt.Printf("DROP MATERIALIZED VIEW %s;\n\n", c.get("matviewname"))
}

// Change handles the case where the names match, but the definition does not
func (c MatViewSchema) Change(obj interface{}) {
	c2, ok := obj.(*MatViewSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a MatViewSchema instance", c2)
	}
	if c.get("definition") != c2.get("definition") {
		fmt.Printf("DROP MATERIALIZED VIEW %s;\n\n", c.get("matviewname"))
		fmt.Printf("CREATE MATERIALIZED VIEW %s AS %s \n\n%s \n\n", c.get("matviewname"), c.get("definition"), c.get("indexdef"))
	}
}

// compareMatViews outputs SQL to make the matviews match between DBs
func compareMatViews(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
	WITH matviews as ( SELECT schemaname || '.' || matviewname AS matviewname,
	definition
	FROM pg_catalog.pg_matviews 
	WHERE schemaname NOT LIKE 'pg_%' 
	)
	SELECT
	matviewname,
	definition,
	COALESCE(string_agg(indexdef, ';' || E'\n\n') || ';', '')  as indexdef
	FROM matviews
	LEFT JOIN  pg_catalog.pg_indexes on matviewname = schemaname || '.' || tablename
	group by matviewname, definition
	ORDER BY
	matviewname;
	`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(MatViewRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(MatViewRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here
	var schema1 Schema = &MatViewSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &MatViewSchema{rows: rows2, rowNum: -1}

	// Compare the matviews
	doDiff(schema1, schema2)
}
