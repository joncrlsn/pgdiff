//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//
// sequence_test.go
package main

import (
	"fmt"
	"github.com/joncrlsn/pgutil"
	"testing"
)

/*
SELECT sequence_schema,
    , {{if eq $.DbSchema "*" }}sequence_schema || '.' || {{end}}sequence_name AS compare_name
    ,  sequence_name AS sequence_name
	, data_type
	, start_value
	, minimum_value
	, maximum_value
	, increment
	, cycle_option
FROM information_schema.sequences
WHERE true
{{if eq $.DbSchema "*" }}
AND sequence_schema NOT LIKE 'pg_%'
AND sequence_schema <> 'information_schema'
{{else}}
AND sequence_schema = '{{$.DbSchema}}'
{{end}}
*/

// Note that these must be sorted by schema and sequence name for this to work
var testSequences1a = []map[string]string{
	{"compare_name": "s1.add", "sequence_schema": "s1", "sequence_name": "s1_add"},
	{"compare_name": "s1.same", "sequence_schema": "s1", "sequence_name": "same"},
	{"compare_name": "s2.add", "sequence_schema": "s2", "sequence_name": "s2_add"},
	{"compare_name": "s2.same", "sequence_schema": "s2", "sequence_name": "same"},
}

// Note that these must be sorted by schema and sequence name for this to work
var testSequences1b = []map[string]string{
	{"compare_name": "s1.delete", "sequence_schema": "s1", "sequence_name": "delete"},
	{"compare_name": "s1.same", "sequence_schema": "s1", "sequence_name": "same"},
	{"compare_name": "s2.same", "sequence_schema": "s2", "sequence_name": "same"},
}

func Test_diffSequencesAllSchemas(t *testing.T) {
	fmt.Println("-- ==========\n-- Sequences all schemas \n-- ==========")
	dbInfo1 = pgutil.DbInfo{DbSchema: "*"}
	dbInfo2 = pgutil.DbInfo{DbSchema: "*"}
	var schema1 Schema = &SequenceSchema{rows: testSequences1a, rowNum: -1}
	var schema2 Schema = &SequenceSchema{rows: testSequences1b, rowNum: -1}
	doDiff(schema1, schema2)
}

// =================================================================================================

// Note that these must be sorted by compare_name (witout schema) for this to work
var testSequences2a = []map[string]string{
	{"compare_name": "add", "sequence_schema": "s1", "sequence_name": "add"},
	{"compare_name": "same", "sequence_schema": "s1", "sequence_name": "same"},
}

// Note that these must be sorted by compare_name (witout schema) for this to work
var testSequences2b = []map[string]string{
	{"compare_name": "delete", "sequence_schema": "s2", "sequence_name": "delete"},
	{"compare_name": "same", "sequence_schema": "s2", "sequence_name": "same"},
}

func Test_diffSequencesBetweenSchemas(t *testing.T) {
	fmt.Println("-- ==========\n-- Sequences between schemas \n-- ==========")
	dbInfo1 = pgutil.DbInfo{DbSchema: "s1"}
	dbInfo2 = pgutil.DbInfo{DbSchema: "s2"}
	var schema1 Schema = &SequenceSchema{rows: testSequences2a, rowNum: -1}
	var schema2 Schema = &SequenceSchema{rows: testSequences2b, rowNum: -1}
	doDiff(schema1, schema2)
}
