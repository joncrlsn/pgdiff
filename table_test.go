//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//
// table_test.go
package main

import (
	"fmt"
	"testing"
)

/*
SELECT table_schema || '.' || table_name AS table_name
    , CASE table_type WHEN 'BASE TABLE' THEN 'TABLE' ELSE table_type END AS table_type
    , is_insertable_into
FROM information_schema.tables
WHERE table_schema NOT LIKE 'pg_%'
WHERE table_schema <> 'information_schema'
AND table_type = 'BASE TABLE'
ORDER BY table_name;
*/

// Note that these must be sorted by table name for this to work
var testTables1a = []map[string]string{
	{"table_name": "schema1.add", "table_type": "TABLE"},
	{"table_name": "schema1.same", "table_type": "TABLE"},
	{"table_name": "schema2.add", "table_type": "TABLE"},
	{"table_name": "schema2.same", "table_type": "TABLE"},
}

// Note that these must be sorted by table_name for this to work
var testTables1b = []map[string]string{
	{"table_name": "schema1.delete", "table_type": "TABLE"},
	{"table_name": "schema1.same", "table_type": "TABLE"},
	{"table_name": "schema2.same", "table_type": "TABLE"},
}

func Test_diffTables(t *testing.T) {
	fmt.Println("-- ==========\n-- Tables\n-- ==========")
	var schema1 Schema = &TableSchema{rows: testTables1a, rowNum: -1}
	var schema2 Schema = &TableSchema{rows: testTables1b, rowNum: -1}
	doDiff(schema1, schema2)
}
