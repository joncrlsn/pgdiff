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
// FunctionRows definition
// ==================================

// FunctionRows is a sortable slice of string maps
type FunctionRows []map[string]string

func (slice FunctionRows) Len() int {
	return len(slice)
}

func (slice FunctionRows) Less(i, j int) bool {
	return slice[i]["function_name"] < slice[j]["function_name"]
}

func (slice FunctionRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// FunctionSchema holds a channel streaming function information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// FunctionSchema implements the Schema interface defined in pgdiff.go
type FunctionSchema struct {
	rows   FunctionRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *FunctionSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *FunctionSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *FunctionSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*FunctionSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a FunctionSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("function_name"), c2.get("function_name"))
	//fmt.Printf("-- Compared %v: %s with %s \n", val, c.get("function_name"), c2.get("function_name"))
	return val
}

// Add returns SQL to create the function
func (c FunctionSchema) Add(obj interface{}) {
	c2, ok := obj.(*FunctionSchema)
	if !ok {
		fmt.Println("Error!!!, FunctionSchema.Add needs a FunctionSchema instance", c2)
	}
	fmt.Println("-- STATEMENT-BEGIN")
	fmt.Println(c.get("definition"))
	fmt.Println("-- STATEMENT-END")
}

// Drop returns SQL to drop the function
func (c FunctionSchema) Drop(obj interface{}) {
	c2, ok := obj.(*FunctionSchema)
	if !ok {
		fmt.Println("Error!!!, FunctionSchema.Drop needs a FunctionSchema instance", c2)
	}
	fmt.Println("-- Note that CASCADE in the statement below will also drop any triggers depending on this function.")
	fmt.Println("-- Also, if there are two functions with this name, you will need to add arguments to identify the correct one to drop.")
	fmt.Println("-- (See http://www.postgresql.org/docs/9.4/interactive/sql-dropfunction.html) ")
	fmt.Printf("DROP FUNCTION %s CASCADE;\n", c.get("function_name"))
}

// Change handles the case where the function names match, but the definition does not
func (c FunctionSchema) Change(obj interface{}) {
	c2, ok := obj.(*FunctionSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a FunctionSchema instance", c2)
	}
	if c.get("definition") != c2.get("definition") {
		fmt.Println("-- This function is different so we'll recreate it:")
		// The definition column has everything needed to rebuild the function
		fmt.Println("-- STATEMENT-BEGIN")
		fmt.Println(c.get("definition"))
		fmt.Println("-- STATEMENT-END")
	}
}

// compareFunctions outputs SQL to make the functions match between DBs
func compareFunctions(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
    SELECT n.nspname || '.' || p.oid::regprocedure   AS function_name
        , t.typname                                  AS return_type
        , pg_get_functiondef(p.oid)                  AS definition
    FROM pg_proc AS p
    JOIN pg_type t ON (p.prorettype = t.oid)
    JOIN pg_namespace n ON (n.oid = p.pronamespace)
    JOIN pg_language l ON (p.prolang = l.oid AND l.lanname IN ('c','plpgsql', 'sql'))
    WHERE n.nspname NOT LIKE 'pg_%';
	`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(FunctionRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(FunctionRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We must explicitly type this as Schema here
	var schema1 Schema = &FunctionSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &FunctionSchema{rows: rows2, rowNum: -1}

	// Compare the functions
	doDiff(schema1, schema2)
}
