//
// Copyright (c) 2014 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"database/sql"
	"fmt"
	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
	"regexp"
	"sort"
	"strings"
)

var curlyBracketRegex = regexp.MustCompile("[{}]")

// RoleRows is a sortable slice of string maps
type RoleRows []map[string]string

func (slice RoleRows) Len() int {
	return len(slice)
}

func (slice RoleRows) Less(i, j int) bool {
	return slice[i]["rolname"] < slice[j]["rolname"]
}

func (slice RoleRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// RoleSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// RoleSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type RoleSchema struct {
	rows   RoleRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *RoleSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *RoleSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *RoleSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *RoleSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*RoleSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a RoleSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("rolname"), c2.get("rolname"))
	return val
}

/*
CREATE ROLE name [ [ WITH ] option [ ... ] ]

where option can be:

      SUPERUSER | NOSUPERUSER
    | CREATEDB | NOCREATEDB
    | CREATEROLE | NOCREATEROLE
    | CREATEUSER | NOCREATEUSER
    | INHERIT | NOINHERIT
    | LOGIN | NOLOGIN
    | REPLICATION | NOREPLICATION
    | CONNECTION LIMIT connlimit
    | [ ENCRYPTED | UNENCRYPTED ] PASSWORD 'password'
    | VALID UNTIL 'timestamp'
    | IN ROLE role_name [, ...]
    | IN GROUP role_name [, ...]
    | ROLE role_name [, ...]
    | ADMIN role_name [, ...]
    | USER role_name [, ...]
    | SYSID uid
*/

// Add generates SQL to add the constraint/index
func (c RoleSchema) Add() {

	// We don't care about efficiency here so we just concat strings
	options := " WITH PASSWORD 'changeme'"

	if c.get("rolcanlogin") == "true" {
		options += " LOGIN"
	} else {
		options += " NOLOGIN"
	}

	if c.get("rolsuper") == "true" {
		options += " SUPERUSER"
	}

	if c.get("rolcreatedb") == "true" {
		options += " CREATEDB"
	}

	if c.get("rolcreaterole") == "true" {
		options += " CREATEROLE"
	}

	if c.get("rolinherit") == "true" {
		options += " INHERIT"
	} else {
		options += " NOINHERIT"
	}

	if c.get("rolreplication") == "true" {
		options += " REPLICATION"
	} else {
		options += " NOREPLICATION"
	}

	if c.get("rolconnlimit") != "-1" && len(c.get("rolconnlimit")) > 0 {
		options += " CONNECTION LIMIT " + c.get("rolconnlimit")
	}
	if c.get("rolvaliduntil") != "null" {
		options += fmt.Sprintf(" VALID UNTIL '%s'", c.get("rolvaliduntil"))
	}

	fmt.Printf("CREATE ROLE %s%s;\n", c.get("rolname"), options)
}

// Drop generates SQL to drop the role
func (c RoleSchema) Drop() {
	fmt.Printf("DROP ROLE %s;\n", c.get("rolname"))
}

// Change handles the case where the role name matches, but the details do not
func (c RoleSchema) Change(obj interface{}) {
	c2, ok := obj.(*RoleSchema)
	if !ok {
		fmt.Println("Error!!!, Change needs a RoleSchema instance", c2)
	}

	options := ""
	if c.get("rolsuper") != c2.get("rolsuper") {
		if c.get("rolsuper") == "true" {
			options += " SUPERUSER"
		} else {
			options += " NOSUPERUSER"
		}
	}

	if c.get("rolcanlogin") != c2.get("rolcanlogin") {
		if c.get("rolcanlogin") == "true" {
			options += " LOGIN"
		} else {
			options += " NOLOGIN"
		}
	}

	if c.get("rolcreatedb") != c2.get("rolcreatedb") {
		if c.get("rolcreatedb") == "true" {
			options += " CREATEDB"
		} else {
			options += " NOCREATEDB"
		}
	}

	if c.get("rolcreaterole") != c2.get("rolcreaterole") {
		if c.get("rolcreaterole") == "true" {
			options += " CREATEROLE"
		} else {
			options += " NOCREATEROLE"
		}
	}

	if c.get("rolcreateuser") != c2.get("rolcreateuser") {
		if c.get("rolcreateuser") == "true" {
			options += " CREATEUSER"
		} else {
			options += " NOCREATEUSER"
		}
	}

	if c.get("rolinherit") != c2.get("rolinherit") {
		if c.get("rolinherit") == "true" {
			options += " INHERIT"
		} else {
			options += " NOINHERIT"
		}
	}

	if c.get("rolreplication") != c2.get("rolreplication") {
		if c.get("rolreplication") == "true" {
			options += " REPLICATION"
		} else {
			options += " NOREPLICATION"
		}
	}

	if c.get("rolconnlimit") != c2.get("rolconnlimit") {
		if len(c.get("rolconnlimit")) > 0 {
			options += " CONNECTION LIMIT " + c.get("rolconnlimit")
		}
	}

	if c.get("rolvaliduntil") != c2.get("rolvaliduntil") {
		if c.get("rolvaliduntil") != "null" {
			options += fmt.Sprintf(" VALID UNTIL '%s'", c.get("rolvaliduntil"))
		}
	}

	// Only alter if we have changes
	if len(options) > 0 {
		fmt.Printf("ALTER ROLE %s%s;\n", c.get("rolname"), options)
	}

	if c.get("memberof") != c2.get("memberof") {
		fmt.Println(c.get("memberof"), "!=", c2.get("memberof"))

		// Remove the curly brackets
		memberof1 := curlyBracketRegex.ReplaceAllString(c.get("memberof"), "")
		memberof2 := curlyBracketRegex.ReplaceAllString(c2.get("memberof"), "")

		// Split
		membersof1 := strings.Split(memberof1, ",")
		membersof2 := strings.Split(memberof2, ",")

		// TODO: Define INHERIT or not
		for _, mo1 := range membersof1 {
			if !misc.ContainsString(membersof2, mo1) {
				fmt.Printf("GRANT %s TO %s;\n", mo1, c.get("rolename"))
			}
		}

		for _, mo2 := range membersof2 {
			if !misc.ContainsString(membersof1, mo2) {
				fmt.Printf("REVOKE %s FROM %s;\n", mo2, c.get("rolename"))
			}
		}

	}
}

/*
 * Compare the roles between two databases or schemas
 */
func compareRoles(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT r.rolname
    , r.rolsuper
    , r.rolinherit
    , r.rolcreaterole
    , r.rolcreatedb
    , r.rolcanlogin
    , r.rolconnlimit
    , r.rolvaliduntil
    , r.rolreplication
	, ARRAY(SELECT b.rolname 
	        FROM pg_catalog.pg_auth_members m  
			JOIN pg_catalog.pg_roles b ON (m.roleid = b.oid)  
	        WHERE m.member = r.oid) as memberof
FROM pg_catalog.pg_roles AS r
ORDER BY r.rolname;
`
	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(RoleRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(RoleRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &RoleSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &RoleSchema{rows: rows2, rowNum: -1}

	// Compare the roles
	doDiff(schema1, schema2)
}
