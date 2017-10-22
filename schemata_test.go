//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//
// schemata_test.go
package main

import (
	"fmt"
	"testing"
)

/*
SELECT schema_name
    , schema_owner
    , default_character_set_schema
FROM information_schema.schemata
WHERE table_schema NOT LIKE 'pg_%'
WHERE table_schema <> 'information_schema'
ORDER BY schema_name;
*/

// Note that these must be sorted by table name for this to work
var testSchematas1 = []map[string]string{
	{"schema_name": "schema_add", "schema_owner": "noop"},
	{"schema_name": "schema_same", "schema_owner": "noop"},
}

// Note that these must be sorted by schema_name for this to work
var testSchematas2 = []map[string]string{
	{"schema_name": "schema_delete", "schema_owner": "noop"},
	{"schema_name": "schema_same", "schema_owner": "noop"},
}

func Test_diffSchematas(t *testing.T) {
	fmt.Println("-- ==========\n-- Schematas\n-- ==========")
	var schema1 Schema = &SchemataSchema{rows: testSchematas1, rowNum: -1}
	var schema2 Schema = &SchemataSchema{rows: testSchematas2, rowNum: -1}
	doDiff(schema1, schema2)
}
