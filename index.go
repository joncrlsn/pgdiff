//
// Copyright (c) 2014 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import "sort"
import "fmt"
import "strings"
import "database/sql"
import "github.com/joncrlsn/pgutil"

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
	if slice[i]["table_name"] == slice[j]["table_name"] {
		return slice[i]["index_name"] < slice[j]["index_name"]
	}
	return slice[i]["table_name"] < slice[j]["table_name"]
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
		fmt.Printf("--Comparing (table_name or index_name is empty): %v\n--           %v\n", c.getRow(), c2.getRow())
	}

	val := _compareString(c.get("table_name"), c2.get("table_name"))
	if val != 0 {
		// Table name differed so return that value
		return val
	}

	// Table name was the same so compare index name
	val = _compareString(c.get("index_name"), c2.get("index_name"))
	return val
}

// Add prints SQL to add the column
func (c *IndexSchema) Add() {
	//fmt.Println("--\n--Add\n--")

	// Assertion
	if c.get("index_def") == "null" || len(c.get("index_def")) == 0 {
		fmt.Printf("-- Add Unexpected situation in index.go: there is no index_def for %s %s\n", c.get("table_name"), c.get("index_name"))
		return
	}

	// Create the index first
	fmt.Printf("%s;\n", c.get("index_def"))

	if c.get("constraint_def") != "null" {
		// Create the constraint using the index we just created
		if c.get("pk") == "true" {
			// Add primary key using the index
			fmt.Printf("ALTER TABLE ONLY %s ADD CONSTRAINT %s PRIMARY KEY USING INDEX %s;\n", c.get("table_name"), c.get("index_name"), c.get("index_name"))
		} else if c.get("uq") == "true" {
			// Add unique constraint using the index
			fmt.Printf("ALTER TABLE ONLY %s ADD CONSTRAINT %s UNIQUE USING INDEX %s;\n", c.get("table_name"), c.get("index_name"), c.get("index_name"))
		}
	}
}

// Drop prints SQL to drop the column
func (c *IndexSchema) Drop() {
	//fmt.Println("--\n--Drop\n--")
	if c.get("constraint_def") != "null" {
		fmt.Println("-- Warning, this may drop foreign keys pointing at this column.  Make sure you re-run the FOREIGN_KEY diff after running this SQL.")
		//fmt.Printf("ALTER TABLE ONLY %s DROP CONSTRAINT IF EXISTS %s CASCADE; -- %s\n", c.get("table_name"), c.get("index_name"), c.get("constraint_def"))
		fmt.Printf("ALTER TABLE ONLY %s DROP CONSTRAINT IF EXISTS %s CASCADE;\n", c.get("table_name"), c.get("index_name"))
	}
	// The second line has no index_def
	//fmt.Printf("DROP INDEX IF EXISTS %s; -- %s \n", c.get("index_name"), c.get("index_def"))
	fmt.Printf("DROP INDEX IF EXISTS %s;\n", c.get("index_name"))
}

// Change handles the case where the table and column match, but the details do not
func (c *IndexSchema) Change(obj interface{}) {
	c2, ok := obj.(*IndexSchema)
	if !ok {
		fmt.Println("-- Error!!!, change needs an IndexSchema instance", c2)
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
			fmt.Printf("DROP INDEX IF EXISTS %s; -- %s \n", c2.get("index_name"), c2.get("index_def"))
		} else if c2.get("constraint_def") == "null" {
			// c1.constraint exists, c2.constraint does not, so
			// Add constraint
			if c.get("index_def") == c2.get("index_def") {
				// Indexes match, so
				// Add constraint using the index
				if c.get("pk") == "true" {
					// Add primary key using the index
					fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY USING INDEX %s;\n", c.get("table_name"), c.get("index_name"), c.get("index_name"))
				} else if c.get("uq") == "true" {
					// Add unique constraint using the index
					fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE USING INDEX %s;\n", c.get("table_name"), c.get("index_name"), c.get("index_name"))
				} else {

				}
			} else {
				// Drop the c2 index, create a copy of the c1 index
				fmt.Printf("DROP INDEX IF EXISTS %s; -- %s \n", c2.get("index_name"), c2.get("index_def"))
			}
			// WIP
			//fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n", c.get("table_name"), c.get("index_name"), c.get("constraint_def"))

		} else if c.get("index_def") != c2.get("index_def") {
			// The constraints match
		}
	} else if c.get("index_def") != c2.get("index_def") {
		if !strings.HasPrefix(c.get("index_def"), c2.get("index_def")) &&
			!strings.HasPrefix(c2.get("index_def"), c.get("index_def")) {
			fmt.Println("--\n--Change index defs different\n--")
			// Remember, if we are here, then the two constraint_defs match (both may be empty)
			// The indexes do not match, but the constraints do
			fmt.Printf("--CHANGE: Different index defs:\n--    %s\n--    %s\n", c.get("index_def"), c2.get("index_def"))

			// Drop the index (and maybe the constraint) so we can recreate the index
			c.Drop()

			// Recreate the index (and a constraint if specified)
			c.Add()
		}
	}

}

// ==================================
// Functions
// ==================================

/*
 * Compare the columns in the two databases
 */
func compareIndexes(conn1 *sql.DB, conn2 *sql.DB) {
	// This SQL was generated with psql -E -c "\d t_org"
	// The "magic" is in pg_get_indexdef and pg_get_constraint
	sql := `
SELECT c.relname AS table_name
    , c2.relname AS index_name
    , i.indisprimary AS pk
    , i.indisunique AS uq
    , pg_catalog.pg_get_indexdef(i.indexrelid, 0, true) AS index_def
    , pg_catalog.pg_get_constraintdef(con.oid, true) AS constraint_def
    , con.contype AS typ
FROM pg_catalog.pg_index AS i
JOIN pg_catalog.pg_class AS c ON (c.oid = i.indrelid)
JOIN pg_catalog.pg_class AS c2 ON (c2.oid = i.indexrelid)
LEFT JOIN pg_catalog.pg_constraint con
    ON (con.conrelid = i.indrelid AND con.conindid = i.indexrelid AND con.contype IN ('p','u','x'))
JOIN pg_catalog.pg_namespace AS n ON  (c2.relnamespace = n.oid)
WHERE c.relname NOT LIKE 'pg_%'
and n.nspname = 'public'
--AND c.relname = 't_org'
--ORDER BY c.relname, c2.relname;
`
	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

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

	// Compare the columns
	doDiff(schema1, schema2)
}
