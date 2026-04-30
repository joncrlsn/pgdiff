//
// Copyright (c) 2026 Hayden Sim.  All rights reserved.
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

// extensions are database-scoped, not schema-scoped, so the dbInfoN.DbSchema
// values are intentionally ignored here.
const extensionSql = `
SELECT e.extname AS extension_name
     , e.extversion AS extension_version
     , n.nspname AS schema_name
FROM pg_catalog.pg_extension e
INNER JOIN pg_catalog.pg_namespace n ON (n.oid = e.extnamespace)
ORDER BY e.extname;
`

// ==================================
// ExtensionRows definition
// ==================================

// ExtensionRows is a sortable slice of string maps
type ExtensionRows []map[string]string

func (slice ExtensionRows) Len() int {
	return len(slice)
}

func (slice ExtensionRows) Less(i, j int) bool {
	return slice[i]["extension_name"] < slice[j]["extension_name"]
}

func (slice ExtensionRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ExtensionSchema holds rows of extension metadata from one of the databases as
// well as a reference to the current row of data we're viewing.
//
// ExtensionSchema implements the Schema interface defined in pgdiff.go
type ExtensionSchema struct {
	rows   ExtensionRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *ExtensionSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *ExtensionSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *ExtensionSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ExtensionSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs an ExtensionSchema instance", c2)
		return +999
	}

	return misc.CompareStrings(c.get("extension_name"), c2.get("extension_name"))
}

// Add returns SQL to create the extension
func (c ExtensionSchema) Add() {
	// schema is preserved from db1; CREATE EXTENSION IF NOT EXISTS keeps the
	// statement idempotent in case the extension was installed out-of-band.
	fmt.Printf(
		"CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s VERSION '%s';\n",
		quoteIdent(c.get("extension_name")),
		quoteIdent(c.get("schema_name")),
		c.get("extension_version"),
	)
}

// Drop returns SQL to drop the extension
func (c ExtensionSchema) Drop() {
	fmt.Printf("DROP EXTENSION IF EXISTS %s;\n", quoteIdent(c.get("extension_name")))
}

// Change handles the case where extension names match but version or schema differ
func (c ExtensionSchema) Change(obj interface{}) {
	c2, ok := obj.(*ExtensionSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs an ExtensionSchema instance", c2)
		return
	}

	if c.get("extension_version") != c2.get("extension_version") {
		fmt.Printf(
			"ALTER EXTENSION %s UPDATE TO '%s';\n",
			quoteIdent(c.get("extension_name")),
			c.get("extension_version"),
		)
	}

	if c.get("schema_name") != c2.get("schema_name") {
		fmt.Printf(
			"ALTER EXTENSION %s SET SCHEMA %s;\n",
			quoteIdent(c.get("extension_name")),
			quoteIdent(c.get("schema_name")),
		)
	}
}

// compareExtensions outputs SQL to make the extensions match between DBs
func compareExtensions(conn1 *sql.DB, conn2 *sql.DB) {

	rowChan1, _ := pgutil.QueryStrings(conn1, extensionSql)
	rowChan2, _ := pgutil.QueryStrings(conn2, extensionSql)

	rows1 := make(ExtensionRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(ExtensionRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We must explicitly type this as Schema here
	var schema1 Schema = &ExtensionSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &ExtensionSchema{rows: rows2, rowNum: -1}

	// Compare the extensions
	doDiff(schema1, schema2)
}
