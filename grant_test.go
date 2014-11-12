// grant_test.go
package main

import (
	"fmt"
	"testing"
)

func Test_parseGrants(t *testing.T) {
	doParseGrants(t, "c42ro=rwa/c42", "c42ro", 3)
	doParseGrants(t, "=arwdDxt/c42", "", 7)
	doParseGrants(t, "user2=arwxt/postgres", "user2", 5)
}

/*
 n.nspname AS schema
  , c.relname AS relationship_name
  , CASE c.relkind
    	WHEN 'r' THEN 'TABLE'
		WHEN 'v' THEN 'VIEW'
		WHEN 'S' THEN 'SEQUENCE'
		WHEN 'f' THEN 'FOREIGN TABLE'
	END as type
  , pg_catalog.array_to_string(c.relacl, E'\n') AS relationship_acl
  , a.attname AS column_name
  , pg_catalog.array_to_string(a.attacl, E'\n') AS column_acl
*/

var test1a = []map[string]string{
	{"schema": "public", "relationship_name": "table1", "type": "TABLE", "relationship_acl": "c42=rwa/postgres", "column_name": "", "column_acl": ""},
	{"schema": "public", "relationship_name": "table1", "type": "TABLE", "relationship_acl": "", "column_name": "column1", "column_acl": "c42ro=rwa/postgres"},
	{"schema": "public", "relationship_name": "table1", "type": "TABLE", "relationship_acl": "", "column_name": "column2", "column_acl": "c42ro=r/postgres\nc42=rwad/postgres"},
	{"schema": "public", "relationship_name": "table2", "type": "TABLE", "relationship_acl": "c42=rwa/postgres", "column_name": "", "column_acl": ""},
}

var test1b = []map[string]string{
	{"schema": "public", "relationship_name": "table1", "type": "TABLE", "relationship_acl": "", "column_name": "column2", "column_acl": "c42ro=r/postgres\nc42=rwad/postgres"},
	{"schema": "public", "relationship_name": "table2", "type": "TABLE", "relationship_acl": "c42=rwad/postgres", "column_name": "t1c1", "column_acl": ""},
}

func Test_diffGrants(t *testing.T) {
	fmt.Println("-- ==========\n-- Grants\n-- ==========")
	var schema1 Schema = &GrantSchema{rows: test1a, rowNum: -1}
	var schema2 Schema = &GrantSchema{rows: test1b, rowNum: -1}
	doDiff(schema1, schema2)
}

func doParseGrants(t *testing.T, acl string, expectedRole string, expectedPermCount int) {
	fmt.Println("Testing", acl)
	role, perms := parseGrants(acl)
	if role != expectedRole {
		t.Error("Wrong role parsed: %s instead of %s", role, expectedRole)
	}
	if len(perms) != expectedPermCount {
		t.Error("Incorrect number of permissions parsed: %d instead of %d", len(perms), expectedPermCount)
	}
}
