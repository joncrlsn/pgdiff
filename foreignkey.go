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
	foreignKeySqlTemplate = initForeignKeySqlTemplate()
)

// Initializes the Sql template
func initForeignKeySqlTemplate() *template.Template {
	sql := `
SELECT {{if eq $.DbSchema "*" }}ns.nspname || '.' || {{end}}cl.relname || '.' || c.conname AS compare_name
    , ns.nspname AS schema_name
	, cl.relname AS table_name
    , c.conname AS fk_name
	, pg_catalog.pg_get_constraintdef(c.oid, true) as constraint_def
FROM pg_catalog.pg_constraint c
INNER JOIN pg_class AS cl ON (c.conrelid = cl.oid)
INNER JOIN pg_namespace AS ns ON (ns.oid = c.connamespace)
WHERE c.contype = 'f'
{{if eq $.DbSchema "*"}}
AND ns.nspname NOT LIKE 'pg_%' 
AND ns.nspname <> 'information_schema' 
{{else}}
AND ns.nspname = '{{$.DbSchema}}'
{{end}}
`
	t := template.New("ForeignKeySqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// ForeignKeyRows definition
// ==================================

// ForeignKeyRows is a sortable string map
type ForeignKeyRows []map[string]string

func (slice ForeignKeyRows) Len() int {
	return len(slice)
}

func (slice ForeignKeyRows) Less(i, j int) bool {
	if slice[i]["compare_name"] != slice[j]["compare_name"] {
		return slice[i]["compare_name"] < slice[j]["compare_name"]
	}
	return slice[i]["constraint_def"] < slice[j]["constraint_def"]
}

func (slice ForeignKeyRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// ForeignKeySchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// ForeignKeySchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type ForeignKeySchema struct {
	rows   ForeignKeyRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *ForeignKeySchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *ForeignKeySchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *ForeignKeySchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *ForeignKeySchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a ForeignKeySchema instance", c2)
		return +999
	}

	//fmt.Printf("Comparing %s with %s", c.get("table_name"), c2.get("table_name"))
	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	if val != 0 {
		return val
	}

	val = misc.CompareStrings(c.get("constraint_def"), c2.get("constraint_def"))
	return val
}

// Add returns SQL to add the foreign key
func (c *ForeignKeySchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}
	fmt.Printf("ALTER TABLE %s.%s ADD CONSTRAINT %s %s;\n", schema, c.get("table_name"), c.get("fk_name"), c.get("constraint_def"))
}

// Drop returns SQL to drop the foreign key
func (c ForeignKeySchema) Drop() {
	fmt.Printf("ALTER TABLE %s.%s DROP CONSTRAINT %s; -- %s\n", c.get("schema_name"), c.get("table_name"), c.get("fk_name"), c.get("constraint_def"))
}

// Change handles the case where the table and foreign key name, but the details do not
func (c *ForeignKeySchema) Change(obj interface{}) {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, ForeignKeySchema.Change(obj) needs a ForeignKeySchema instance", c2)
	}
	// There is no "changing" a foreign key.  It either gets created or dropped (or left as-is).
}

/*
 * Compare the foreign keys in the two databases.
 */
func compareForeignKeys(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	foreignKeySqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	foreignKeySqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(ForeignKeyRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(ForeignKeyRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &ForeignKeySchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &ForeignKeySchema{rows: rows2, rowNum: -1}

	// Compare the columns
	doDiff(schema1, schema2)
}
