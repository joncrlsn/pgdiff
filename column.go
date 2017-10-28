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
	"strconv"
	"strings"
	"text/template"
)

var (
	columnSqlTemplate = initColumnSqlTemplate()
)

// Initializes the Sql template
func initColumnSqlTemplate() *template.Template {
	sql := `
SELECT table_schema
    , {{if eq $.DbSchema "*" }}table_schema || '.' || {{end}}table_name AS compare_name
	, table_name
    , column_name
    , data_type
    , is_nullable
    , column_default
    , character_maximum_length
FROM information_schema.columns 
WHERE is_updatable = 'YES'
{{if eq $.DbSchema "*" }}
AND table_schema NOT LIKE 'pg_%' 
AND table_schema <> 'information_schema' 
{{else}}
AND table_schema = '{{$.DbSchema}}'
{{end}}
ORDER BY compare_name, column_name;
`
	t := template.New("ColumnSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// Column Rows definition
// ==================================

// ColumnRows is a sortable slice of string maps
type ColumnRows []map[string]string

func (slice ColumnRows) Len() int {
	return len(slice)
}

func (slice ColumnRows) Less(i, j int) bool {
	if slice[i]["compare_name"] != slice[j]["compare_name"] {
		return slice[i]["column_name"] < slice[j]["compare_name"]
	}
	return slice[i]["column_name"] < slice[j]["column_name"]
}

func (slice ColumnRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// ColumnSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// ColumnSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type ColumnSchema struct {
	rows   ColumnRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *ColumnSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *ColumnSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *ColumnSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ColumnSchema)
	if !ok {
		fmt.Println("Error!!!, Compare needs a ColumnSchema instance", c2)
	}

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	if val != 0 {
		// Table name differed so return that value
		return val
	}

	// Table name was the same so compare column name
	val = misc.CompareStrings(c.get("column_name"), c2.get("column_name"))
	return val
}

// Add prints SQL to add the column
func (c *ColumnSchema) Add() {

	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("table_schema")
	}

	if c.get("data_type") == "character varying" {
		maxLength, valid := getMaxLength(c.get("character_maximum_length"))
		if !valid {
			fmt.Printf("ALTER TABLE %s.%s ADD COLUMN %s character varying", schema, c.get("table_name"), c.get("column_name"))
		} else {
			fmt.Printf("ALTER TABLE %s.%s ADD COLUMN %s character varying(%s)", schema, c.get("table_name"), c.get("column_name"), maxLength)
		}
	} else {
		if c.get("data_type") == "ARRAY" {
			fmt.Println("-- Note that adding of array data types are not yet generated properly.")
		}
		fmt.Printf("ALTER TABLE %s.%s ADD COLUMN %s %s", schema, c.get("table_name"), c.get("column_name"), c.get("data_type"))
	}

	if c.get("is_nullable") == "NO" {
		fmt.Printf(" NOT NULL")
	}
	if c.get("column_default") != "null" {
		fmt.Printf(" DEFAULT %s", c.get("column_default"))
	}
	fmt.Printf(";\n")
}

// Drop prints SQL to drop the column
func (c *ColumnSchema) Drop() {
	// if dropping column
	fmt.Printf("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;\n", c.get("table_schema"), c.get("table_name"), c.get("column_name"))
}

// Change handles the case where the table and column match, but the details do not
func (c *ColumnSchema) Change(obj interface{}) {
	c2, ok := obj.(*ColumnSchema)
	if !ok {
		fmt.Println("Error!!!, ColumnSchema.Change(obj) needs a ColumnSchema instance", c2)
	}

	// Detect column type change (mostly varchar length, or number size increase)
	// (integer to/from bigint is OK)
	if c.get("data_type") == c2.get("data_type") {
		if c.get("data_type") == "character varying" {
			max1, max1Valid := getMaxLength(c.get("character_maximum_length"))
			max2, max2Valid := getMaxLength(c2.get("character_maximum_length"))
			if !max1Valid && !max2Valid {
				// Leave them alone, they both have undefined max lengths
			} else if (max1Valid || !max2Valid) && (max1 != c2.get("character_maximum_length")) {
				//if !max1Valid {
				//    fmt.Println("-- WARNING: varchar column has no maximum length.  Setting to 1024, which may result in data loss.")
				//}
				max1Int, err1 := strconv.Atoi(max1)
				check("converting string to int", err1)
				max2Int, err2 := strconv.Atoi(max2)
				check("converting string to int", err2)
				if max1Int < max2Int {
					fmt.Println("-- WARNING: The next statement will shorten a character varying column, which may result in data loss.")
				}
				fmt.Printf("-- max1Valid: %v  max2Valid: %v \n", max1Valid, max2Valid)
				fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE character varying(%s);\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"), max1)
			}
		}
	}

	// Code and test a column change from integer to bigint
	if c.get("data_type") != c2.get("data_type") {
		fmt.Printf("-- WARNING: This type change may not work well: (%s to %s).\n", c2.get("data_type"), c.get("data_type"))
		if strings.HasPrefix(c.get("data_type"), "character") {
			max1, max1Valid := getMaxLength(c.get("character_maximum_length"))
			if !max1Valid {
				fmt.Println("-- WARNING: varchar column has no maximum length.  Setting to 1024")
			}
			fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s(%s);\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"), c.get("data_type"), max1)
		} else {
			fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s;\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"), c.get("data_type"))
		}
	}

	// Detect column default change (or added, dropped)
	if c.get("column_default") == "null" {
		if c.get("column_default") != "null" {
			fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s DROP DEFAULT;\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"))
		}
	} else if c.get("column_default") != c2.get("column_default") {
		fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s SET DEFAULT %s;\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"), c.get("column_default"))
	}

	// Detect not-null and nullable change
	if c.get("is_nullable") != c2.get("is_nullable") {
		if c.get("is_nullable") == "YES" {
			fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s DROP NOT NULL;\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"))
		} else {
			fmt.Printf("ALTER TABLE %s.%s ALTER COLUMN %s SET NOT NULL;\n", c2.get("table_schema"), c.get("table_name"), c.get("column_name"))
		}
	}
}

// ==================================
// Standalone Functions
// ==================================

// compareColumns outputs SQL to make the columns match between two databases or schemas
func compareColumns(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	columnSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	columnSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	//rows1 := make([]map[string]string, 500)
	rows1 := make(ColumnRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	//rows2 := make([]map[string]string, 500)
	rows2 := make(ColumnRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(&rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &ColumnSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &ColumnSchema{rows: rows2, rowNum: -1}

	// Compare the columns
	doDiff(schema1, schema2)
}

// getMaxLength returns the maximum length and whether or not it is valid
func getMaxLength(maxLength string) (string, bool) {

	if maxLength == "null" {
		// default to 1024
		return "1024", false
	}
	return maxLength, true
}
