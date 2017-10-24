//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
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
// SchemataRows definition
// ==================================

// SchemataRows is a sortable slice of string maps
type SchemataRows []map[string]string

func (slice SchemataRows) Len() int {
	return len(slice)
}

func (slice SchemataRows) Less(i, j int) bool {
	return slice[i]["schema_name"] < slice[j]["schema_name"]
}

func (slice SchemataRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// SchemataSchema holds a channel streaming schema meta information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// SchemataSchema implements the Schema interface defined in pgdiff.go
type SchemataSchema struct {
	rows   SchemataRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *SchemataSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *SchemataSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *SchemataSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*SchemataSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a SchemataSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("schema_name"), c2.get("schema_name"))
	//fmt.Printf("-- Compared %v: %s with %s \n", val, c.get("schema_name"), c2.get("schema_name"))
	return val
}

// Add returns SQL to add the schemata
func (c SchemataSchema) Add() {
	// CREATE SCHEMA schema_name [ AUTHORIZATION user_name
	fmt.Printf("CREATE SCHEMA %s AUTHORIZATION %s;", c.get("schema_name"), c.get("schema_owner"))
	fmt.Println()
}

// Drop returns SQL to drop the schemata
func (c SchemataSchema) Drop() {
	// DROP SCHEMA [ IF EXISTS ] name [, ...] [ CASCADE | RESTRICT ]
	fmt.Printf("DROP SCHEMA IF EXISTS %s;\n", c.get("schema_name"))
}

// Change handles the case where the schema name matches, but the details do not
func (c SchemataSchema) Change(obj interface{}) {
	c2, ok := obj.(*SchemataSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a SchemataSchema instance", c2)
	}
	// There's nothing we need to do here
}

// compareSchematas outputs SQL to make the schema names match between DBs
func compareSchematas(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT schema_name
    , schema_owner
    , default_character_set_schema
FROM information_schema.schemata
WHERE schema_name NOT LIKE 'pg_%' 
  AND schema_name <> 'information_schema' 
ORDER BY schema_name;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(SchemataRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(SchemataRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here
	var schema1 Schema = &SchemataSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &SchemataSchema{rows: rows2, rowNum: -1}

	// Compare the tables
	doDiff(schema1, schema2)
}
