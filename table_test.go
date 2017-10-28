//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//
// table_test.go
package main

import (
	"fmt"
	"github.com/joncrlsn/pgutil"
	"testing"
)

/*
SELECT table_schema
    , {{if eq $.DbSchema "*" }}table_schema || '.' || {{end}}table_name AS compare_name
	, table_name
    , CASE table_type
	  WHEN 'BASE TABLE' THEN 'TABLE'
	  ELSE table_type END AS table_type
    , is_insertable_into
FROM information_schema.tables
WHERE table_type = 'BASE TABLE'
{{if eq $.DbSchema "*" }}
AND table_schema NOT LIKE 'pg_%'
AND table_schema <> 'information_schema'
{{else}}
AND table_schema = '{{$.DbSchema}}'
{{end}}
ORDER BY compare_name;
*/

// Note that these must be sorted by schema and table name for this to work
var testTables1a = []map[string]string{
	{"compare_name": "s1.add", "table_schema": "s1", "table_name": "s1_add", "table_type": "TABLE"},
	{"compare_name": "s1.same", "table_schema": "s1", "table_name": "same", "table_type": "TABLE"},
	{"compare_name": "s2.add", "table_schema": "s2", "table_name": "s2_add", "table_type": "TABLE"},
	{"compare_name": "s2.same", "table_schema": "s2", "table_name": "same", "table_type": "TABLE"},
}

// Note that these must be sorted by schema and table name for this to work
var testTables1b = []map[string]string{
	{"compare_name": "s1.delete", "table_schema": "s1", "table_name": "delete", "table_type": "TABLE"},
	{"compare_name": "s1.same", "table_schema": "s1", "table_name": "same", "table_type": "TABLE"},
	{"compare_name": "s2.same", "table_schema": "s2", "table_name": "same", "table_type": "TABLE"},
}

func Test_diffTablesAllSchemas(t *testing.T) {
	fmt.Println("-- ==========\n-- Tables all schemas \n-- ==========")
	dbInfo1 = pgutil.DbInfo{DbSchema: "*"}
	dbInfo2 = pgutil.DbInfo{DbSchema: "*"}
	var schema1 Schema = &TableSchema{rows: testTables1a, rowNum: -1}
	var schema2 Schema = &TableSchema{rows: testTables1b, rowNum: -1}
	doDiff(schema1, schema2)
}

// =================================================================================================

// Note that these must be sorted by compare_name (witout schema) for this to work
var testTables2a = []map[string]string{
	{"compare_name": "add", "table_schema": "s1", "table_name": "add", "table_type": "TABLE"},
	{"compare_name": "same", "table_schema": "s1", "table_name": "same", "table_type": "TABLE"},
}

// Note that these must be sorted by compare_name (witout schema) for this to work
var testTables2b = []map[string]string{
	{"compare_name": "delete", "table_schema": "s2", "table_name": "delete", "table_type": "TABLE"},
	{"compare_name": "same", "table_schema": "s2", "table_name": "same", "table_type": "TABLE"},
}

func Test_diffTablesBetweenSchemas(t *testing.T) {
	fmt.Println("-- ==========\n-- Tables between schemas \n-- ==========")
	dbInfo1 = pgutil.DbInfo{DbSchema: "s1"}
	dbInfo2 = pgutil.DbInfo{DbSchema: "s2"}
	var schema1 Schema = &TableSchema{rows: testTables2a, rowNum: -1}
	var schema2 Schema = &TableSchema{rows: testTables2b, rowNum: -1}
	doDiff(schema1, schema2)
}
