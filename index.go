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
	"strings"
	"text/template"
)

var (
	indexSqlTemplate = initIndexSqlTemplate()
)

// Initializes the Sql template
func initIndexSqlTemplate() *template.Template {
	sql := `
SELECT {{if eq $.DbSchema "*" }}n.nspname || '.' || {{end}}c.relname || '.' || c2.relname AS compare_name
    , n.nspname AS schema_name
    , c.relname AS table_name
    , c2.relname AS index_name
    , i.indisprimary AS pk
    , i.indisunique AS uq
    , pg_catalog.pg_get_indexdef(i.indexrelid, 0, true) AS index_def
    , pg_catalog.pg_get_constraintdef(con.oid, true) AS constraint_def
    , con.contype AS typ
FROM pg_catalog.pg_index AS i
INNER JOIN pg_catalog.pg_class AS c ON (c.oid = i.indrelid)
INNER JOIN pg_catalog.pg_class AS c2 ON (c2.oid = i.indexrelid)
LEFT OUTER JOIN pg_catalog.pg_constraint con
    ON (con.conrelid = i.indrelid AND con.conindid = i.indexrelid AND con.contype IN ('p','u','x'))
INNER JOIN pg_catalog.pg_namespace AS n ON (c2.relnamespace = n.oid)
WHERE true
{{if eq $.DbSchema "*"}}
AND n.nspname NOT LIKE 'pg_%' 
AND n.nspname <> 'information_schema' 
{{else}}
AND n.nspname = '{{$.DbSchema}}'
{{end}}
`
	t := template.New("IndexSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// IndexRows definition
// ==================================

// IndexRows is a sortable slice of string maps
type IndexRows []map[string]string

func (slice IndexRows) Len() int {
	return len(slice)
}

func (slice IndexRows) Less(i, j int) bool {
	//fmt.Printf("--Less %s:%s with %s:%s", slice[i]["table_name"], slice[i]["column_name"], slice[j]["table_name"], slice[j]["column_name"])
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}

func (slice IndexRows) Swap(i, j int) {
	//fmt.Printf("--Swapping %d/%s:%s with %d/%s:%s \n", i, slice[i]["table_name"], slice[i]["index_name"], j, slice[j]["table_name"], slice[j]["index_name"])
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// IndexSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// IndexSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type IndexSchema struct {
	rows   IndexRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *IndexSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *IndexSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *IndexSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *IndexSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*IndexSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a IndexSchema instance", c2)
		return +999
	}

	if len(c.get("table_name")) == 0 || len(c.get("index_name")) == 0 {
		fmt.Printf("--Comparing (table_name and/or index_name is empty): %v\n", c.getRow())
		fmt.Printf("--           %v\n", c2.getRow())
	}

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	return val
}

// Add prints SQL to add the index
func (c *IndexSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}

	// Assertion
	if c.get("index_def") == "null" || len(c.get("index_def")) == 0 {
		fmt.Printf("-- Add Unexpected situation in index.go: there is no index_def for %s.%s %s\n", schema, c.get("table_name"), c.get("index_name"))
		return
	}

	// If we are comparing two different schemas against each other, we need to do some
	// modification of the first index_def so we create the index in the write schema
	indexDef := c.get("index_def")
	if dbInfo1.DbSchema != dbInfo2.DbSchema {
		indexDef = strings.Replace(
			indexDef,
			fmt.Sprintf(" %s.%s ", c.get("schema_name"), c.get("table_name")),
			fmt.Sprintf(" %s.%s ", dbInfo2.DbSchema, c.get("table_name")),
			-1)
	}

	fmt.Println(indexDef)

	if c.get("constraint_def") != "null" {
		// Create the constraint using the index we just created
		if c.get("pk") == "true" {
			// Add primary key using the index
			fmt.Printf("ALTER TABLE %s.%s ADD CONSTRAINT %s PRIMARY KEY USING INDEX %s; -- (1)\n", schema, c.get("table_name"), c.get("index_name"), c.get("index_name"))
		} else if c.get("uq") == "true" {
			// Add unique constraint using the index
			fmt.Printf("ALTER TABLE %s.%s ADD CONSTRAINT %s UNIQUE USING INDEX %s; -- (2)\n", schema, c.get("table_name"), c.get("index_name"), c.get("index_name"))
		}
	}
}

// Drop prints SQL to drop the index
func (c *IndexSchema) Drop() {
	if c.get("constraint_def") != "null" {
		fmt.Println("-- Warning, this may drop foreign keys pointing at this column.  Make sure you re-run the FOREIGN_KEY diff after running this SQL.")
		fmt.Printf("ALTER TABLE %s.%s DROP CONSTRAINT %s CASCADE; -- %s\n", c.get("schema_name"), c.get("table_name"), c.get("index_name"), c.get("constraint_def"))
	}
	fmt.Printf("DROP INDEX %s.%s;\n", c.get("schema_name"), c.get("index_name"))
}

// Change handles the case where the table and column match, but the details do not
func (c *IndexSchema) Change(obj interface{}) {
	c2, ok := obj.(*IndexSchema)
	if !ok {
		fmt.Println("-- Error!!!, Change needs an IndexSchema instance", c2)
	}

	// Table and constraint name matches... We need to make sure the details match

	// NOTE that there should always be an index_def for both c and c2 (but we're checking below anyway)
	if len(c.get("index_def")) == 0 {
		fmt.Printf("-- Change: Unexpected situation in index.go: index_def is empty for 1: %v  2:%v\n", c.getRow(), c2.getRow())
		return
	}
	if len(c2.get("index_def")) == 0 {
		fmt.Printf("-- Change: Unexpected situation in index.go: index_def is empty for 2: %v 1: %v\n", c2.getRow(), c.getRow())
		return
	}

	if c.get("constraint_def") != c2.get("constraint_def") {
		// c1.constraint and c2.constraint are just different
		fmt.Printf("-- CHANGE: Different defs:\n--    %s\n--    %s\n", c.get("constraint_def"), c2.get("constraint_def"))
		if c.get("constraint_def") == "null" {
			// c1.constraint does not exist, c2.constraint does, so
			// Drop constraint
			fmt.Printf("DROP INDEX %s; -- %s \n", c2.get("index_name"), c2.get("index_def"))
		} else if c2.get("constraint_def") == "null" {
			// c1.constraint exists, c2.constraint does not, so
			// Add constraint
			if c.get("index_def") == c2.get("index_def") {
				// Indexes match, so
				// Add constraint using the index
				if c.get("pk") == "true" {
					// Add primary key using the index
					fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY USING INDEX %s; -- (3)\n", c.get("table_name"), c.get("index_name"), c.get("index_name"))
				} else if c.get("uq") == "true" {
					// Add unique constraint using the index
					fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE USING INDEX %s; -- (4)\n", c.get("table_name"), c.get("index_name"), c.get("index_name"))
				} else {

				}
			} else {
				// Drop the c2 index, create a copy of the c1 index
				fmt.Printf("DROP INDEX %s; -- %s \n", c2.get("index_name"), c2.get("index_def"))
			}
			// WIP
			//fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n", c.get("table_name"), c.get("index_name"), c.get("constraint_def"))

		} else if c.get("index_def") != c2.get("index_def") {
			// The constraints match
		}

		return
	}

	// At this point, we know that the constraint_def matches.  Compare the index_def

	indexDef1 := c.get("index_def")
	indexDef2 := c2.get("index_def")

	// If we are comparing two different schemas against each other, we need to do
	// some modification of the first index_def so it looks more like the second
	if dbInfo1.DbSchema != dbInfo2.DbSchema {
		indexDef1 = strings.Replace(
			indexDef1,
			fmt.Sprintf(" %s.%s ", c.get("schema_name"), c.get("table_name")),
			fmt.Sprintf(" %s.%s ", c2.get("schema_name"), c2.get("table_name")),
			-1,
		)
	}

	if indexDef1 != indexDef2 {
		// Notice that, if we are here, then the two constraint_defs match (both may be empty)
		// The indexes do not match, but the constraints do
		if !strings.HasPrefix(c.get("index_def"), c2.get("index_def")) &&
			!strings.HasPrefix(c2.get("index_def"), c.get("index_def")) {
			fmt.Println("--\n--CHANGE: index defs are different for identical constraint defs:")
			fmt.Printf("--    %s\n--    %s\n", c.get("index_def"), c2.get("index_def"))

			// Drop the index (and maybe the constraint) so we can recreate the index
			c.Drop()

			// Recreate the index (and a constraint if specified)
			c.Add()
		}
	}

}

// compareIndexes outputs Sql to make the indexes match between to DBs or schemas
func compareIndexes(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	indexSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	indexSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(IndexRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(IndexRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &IndexSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &IndexSchema{rows: rows2, rowNum: -1}

	// Compare the indexes
	doDiff(schema1, schema2)
}
