// grant_test.go
package main

import (
	"fmt"
	"testing"
)

/*
SELECT r.rolname
    , r.rolsuper
    , r.rolinherit
    , r.rolcreaterole
    , r.rolcreatedb
    , r.rolcanlogin
    , r.rolconnlimit
    , r.rolvaliduntil
    , r.rolreplication
FROM pg_catalog.pg_roles AS r
ORDER BY r.rolname;
*/

// Note that these must be sorted by rolname for this to work
var testRoles1a = []map[string]string{
	{"rolname": "addme2", "rolsuper": "false", "rolinherit": "false", "rolcreaterole": "true", "rolcreatedb": "true", "rolcanlogin": "true", "rolconnlimit": "100", "rolvaliduntil": "null"},
	{"rolname": "changeme", "rolsuper": "false", "rolinherit": "false", "rolcreaterole": "true", "rolcreatedb": "true", "rolcanlogin": "true", "rolconnlimit": "100", "rolvaliduntil": "null"},
	{"rolname": "matchme", "rolsuper": "false", "rolinherit": "false", "rolcreaterole": "true", "rolcreatedb": "true", "rolcanlogin": "true", "rolconnlimit": "100", "rolvaliduntil": "null"},
	{"rolname": "x-addme1", "rolsuper": "true", "rolinherit": "false", "rolcreaterole": "true", "rolcreatedb": "true", "rolcanlogin": "true", "rolconnlimit": "-1", "rolvaliduntil": "null"},
}

// Note that these must be sorted by rolname for this to work
var testRoles1b = []map[string]string{
	{"rolname": "changeme", "rolsuper": "false", "rolinherit": "false", "rolcreaterole": "false", "rolcreatedb": "false", "rolcanlogin": "true", "rolconnlimit": "10", "rolvaliduntil": "null"},
	{"rolname": "deleteme"},
	{"rolname": "matchme", "rolsuper": "false", "rolinherit": "false", "rolcreaterole": "true", "rolcreatedb": "true", "rolcanlogin": "true", "rolconnlimit": "100", "rolvaliduntil": "null"},
}

func Test_diffRoles(t *testing.T) {
	fmt.Println("-- ==========\n-- Roles\n-- ==========")
	var schema1 Schema = &RoleSchema{rows: testRoles1a, rowNum: -1}
	var schema2 Schema = &RoleSchema{rows: testRoles1b, rowNum: -1}
	doDiff(schema1, schema2)
}
