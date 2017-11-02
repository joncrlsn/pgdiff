//
// Copyright (c) 2016 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"database/sql"
	"fmt"
	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
	"text/template"
	"bytes"
	"sort"
)

var (
	triggerSqlTemplate = initTriggerSqlTemplate()
)

// Initializes the Sql template
func initTriggerSqlTemplate() *template.Template {
	sql := `
    SELECT n.nspname AS schema_name
       , {{if eq $.DbSchema "*" }}n.nspname || '.' || {{end}}c.relname AS compare_name
       , c.relname AS table_name
       , t.tgname AS trigger_name
       , pg_catalog.pg_get_triggerdef(t.oid, true) AS definition
       , t.tgenabled AS enabled
    FROM pg_catalog.pg_trigger t
    INNER JOIN pg_catalog.pg_class c ON (c.oid = t.tgrelid)
    INNER JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	WHERE true
    {{if eq $.DbSchema "*" }}
    AND n.nspname NOT LIKE 'pg_%' 
    AND n.nspname <> 'information_schema' 
    {{else}}
    AND n.nspname = '{{$.DbSchema}}'
    {{end}}
	`
	t := template.New("TriggerSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// TriggerRows definition
// ==================================

// TriggerRows is a sortable slice of string maps
type TriggerRows []map[string]string

func (slice TriggerRows) Len() int {
	return len(slice)
}

func (slice TriggerRows) Less(i, j int) bool {
	if slice[i]["compare_name"] != slice[j]["compare_name"] {
		return slice[i]["compare_name"] < slice[j]["compare_name"]
	}
	return slice[i]["trigger_name"] < slice[j]["trigger_name"]
}

func (slice TriggerRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// TriggerSchema holds a channel streaming trigger information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// TriggerSchema implements the Schema interface defined in pgdiff.go
type TriggerSchema struct {
	rows   TriggerRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *TriggerSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *TriggerSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *TriggerSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*TriggerSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a TriggerSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	if val != 0 {
		return val
	}
	val = misc.CompareStrings(c.get("trigger_name"), c2.get("trigger_name"))
	return val
}

// Add returns SQL to create the trigger
func (c TriggerSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}

	// NOTE: we may need to do some tweaking of the definition here to replace the old schema
	// name with the new schema name.

	fmt.Printf("%s;\n", c.get("definition"))
}

// Drop returns SQL to drop the trigger
func (c TriggerSchema) Drop() {
	fmt.Printf("DROP TRIGGER %s ON %s.%s;\n", c.get("trigger_name"), c.get("schema_name"), c.get("table_name"))
}

// Change handles the case where the trigger names match, but the definition does not
func (c TriggerSchema) Change(obj interface{}) {
	c2, ok := obj.(*TriggerSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a TriggerSchema instance", c2)
	}
	if c.get("definition") != c2.get("definition") {
		fmt.Println("-- This function looks different so we'll recreate it:")
		// The definition column has everything needed to rebuild the function
		fmt.Println("-- STATEMENT-BEGIN")
		fmt.Println(c.get("definition"))
		fmt.Println("-- STATEMENT-END")
	}
}

// compareTriggers outputs SQL to make the triggers match between DBs
func compareTriggers(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	triggerSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	triggerSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(TriggerRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(TriggerRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We must explicitly type this as Schema here
	var schema1 Schema = &TriggerSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &TriggerSchema{rows: rows2, rowNum: -1}

	// Compare the triggers
	doDiff(schema1, schema2)
}
