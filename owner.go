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
	ownerSqlTemplate = initOwnerSqlTemplate()
)

// Initializes the Sql template
func initOwnerSqlTemplate() *template.Template {
	sql := `
SELECT n.nspname AS schema_name
    , {{if eq $.DbSchema "*" }}n.nspname || '.' || {{end}}c.relname || '.' || c.relname AS compare_name
    , c.relname AS relationship_name
    , a.rolname AS owner
    , CASE WHEN c.relkind = 'r' THEN 'TABLE' 
        WHEN c.relkind = 'S' THEN 'SEQUENCE' 
        WHEN c.relkind = 'v' THEN 'VIEW' 
        ELSE c.relkind::varchar END AS type
FROM pg_class AS c
INNER JOIN pg_authid AS a ON (a.oid = c.relowner)
INNER JOIN pg_namespace AS n ON (n.oid = c.relnamespace)
WHERE c.relkind IN ('r', 'S', 'v')
{{if eq $.DbSchema "*" }}
AND n.nspname NOT LIKE 'pg_%' 
AND n.nspname <> 'information_schema'
{{else}}
AND n.nspname = '{{$.DbSchema}}'
{{end}}
;`

	t := template.New("OwnerSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// OwnerRows definition
// ==================================

// OwnerRows is a sortable slice of string maps
type OwnerRows []map[string]string

func (slice OwnerRows) Len() int {
	return len(slice)
}

func (slice OwnerRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}

func (slice OwnerRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// OwnerSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// OwnerSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type OwnerSchema struct {
	rows   OwnerRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *OwnerSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *OwnerSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *OwnerSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *OwnerSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*OwnerSchema)
	if !ok {
		fmt.Println("Error!!!, Compare needs a OwnerSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	return val
}

// Add generates SQL to add the table/view owner
func (c OwnerSchema) Add() {
	fmt.Printf("-- Notice!, db2 has no %s named %s.  First, run pgdiff with the %s option.\n", c.get("type"), c.get("relationship_name"), c.get("type"))
}

// Drop generates SQL to drop the owner
func (c OwnerSchema) Drop() {
	fmt.Printf("-- Notice!, db2 has a %s that db1 does not: %s.   First, run pgdiff with the %s option.\n", c.get("type"), c.get("relationship_name"), c.get("type"))
}

// Change handles the case where the relationship name matches, but the owner does not
func (c OwnerSchema) Change(obj interface{}) {
	c2, ok := obj.(*OwnerSchema)
	if !ok {
		fmt.Println("-- Error!!!, Change needs a OwnerSchema instance", c2)
	}

	if c.get("owner") != c2.get("owner") {
		fmt.Printf("ALTER %s %s.%s OWNER TO %s; \n", c.get("type"), c2.get("schema_name"), c.get("relationship_name"), c.get("owner"))
	}
}

/*
 * Compare the ownership of tables, sequences, and views between two databases or schemas
 */
func compareOwners(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	ownerSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	ownerSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(OwnerRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(OwnerRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &OwnerSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &OwnerSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}
