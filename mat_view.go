//
// Copyright (c) 2016 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"sort"
	"text/template"

	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
)

var (
	matViewSqlTemplate = initMatViewSqlTemplate()
)

// initMatViewSqlTemplate builds the query used to list matviews, optionally scoped to a single schema.
func initMatViewSqlTemplate() *template.Template {
	sql := `
WITH matviews AS (
	SELECT
		schemaname,
		matviewname,
		schemaname || '.' || matviewname AS qualname,
		definition
	FROM pg_catalog.pg_matviews
	WHERE 1=1
	AND NOT EXISTS (
		SELECT 1
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_depend d ON d.classid = 'pg_class'::regclass
		                AND d.objid = c.oid
		                AND d.deptype = 'e'
		WHERE n.nspname = schemaname AND c.relname = matviewname
	)
	{{if eq $.DbSchema "*" }}
	AND schemaname NOT LIKE 'pg_%'
	{{else}}
	AND schemaname = '{{$.DbSchema}}'
	{{end}}
)
SELECT
	mv.qualname AS matviewname,
	mv.schemaname AS schema_name,
	mv.matviewname AS mat_view_name,
	mv.definition,
	COALESCE(string_agg(i.indexdef, ';' || E'\n\n') || ';', '') AS indexdef
FROM matviews mv
LEFT JOIN pg_catalog.pg_indexes i ON mv.qualname = i.schemaname || '.' || i.tablename
GROUP BY mv.qualname, mv.schemaname, mv.matviewname, mv.definition
ORDER BY mv.qualname;
`
	t := template.New("MatViewSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

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

// matViewRef returns the quoted schema.matview identifier for the current row.
func (c *MatViewSchema) matViewRef() string {
	return quoteQualified(c.get("schema_name"), c.get("mat_view_name"))
}

// Add returns SQL to create the matview
func (c MatViewSchema) Add() {
	fmt.Printf("CREATE MATERIALIZED VIEW %s AS %s \n\n%s \n\n", c.matViewRef(), c.get("definition"), c.get("indexdef"))
}

// Drop returns SQL to drop the matview
func (c MatViewSchema) Drop() {
	fmt.Printf("DROP MATERIALIZED VIEW %s;\n\n", c.matViewRef())
}

// Change handles the case where the names match, but the definition does not
func (c MatViewSchema) Change(obj interface{}) {
	c2, ok := obj.(*MatViewSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a MatViewSchema instance", c2)
	}
	if !defsEqual(c.get("definition"), c2.get("definition")) {
		fmt.Printf("DROP MATERIALIZED VIEW %s;\n\n", c.matViewRef())
		fmt.Printf("CREATE MATERIALIZED VIEW %s AS %s \n\n%s \n\n", c.matViewRef(), c.get("definition"), c.get("indexdef"))
	}
}

// compareMatViews outputs SQL to make the matviews match between DBs
func compareMatViews(conn1 *sql.DB, conn2 *sql.DB) {
	buf1 := new(bytes.Buffer)
	matViewSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	matViewSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

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
