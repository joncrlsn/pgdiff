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
	"strings"
	"text/template"
)

var (
	grantRelationshipSqlTemplate = initGrantRelationshipSqlTemplate()
)

// Initializes the Sql template
func initGrantRelationshipSqlTemplate() *template.Template {
	sql := `
SELECT n.nspname AS schema_name
  , {{ if eq $.DbSchema "*" }}n.nspname || '.' || {{ end }}c.relkind || '.' || c.relname AS compare_name
  , CASE c.relkind
    WHEN 'r' THEN 'TABLE'
    WHEN 'v' THEN 'VIEW'
    WHEN 'S' THEN 'SEQUENCE'
    WHEN 'f' THEN 'FOREIGN TABLE'
    END as type
  , c.relname AS relationship_name
  , unnest(c.relacl) AS relationship_acl
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
WHERE c.relkind IN ('r', 'v', 'S', 'f')
AND pg_catalog.pg_table_is_visible(c.oid)
{{ if eq $.DbSchema "*" }}
AND n.nspname NOT LIKE 'pg_%'
AND n.nspname <> 'information_schema'
{{ else }}
AND n.nspname = '{{ $.DbSchema }}'
{{ end }};
ORDER BY n.nspname, c.relname;
`

	t := template.New("GrantAttributeSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// GrantRelationshipRows definition
// ==================================

// GrantRelationshipRows is a sortable slice of string maps
type GrantRelationshipRows []map[string]string

func (slice GrantRelationshipRows) Len() int {
	return len(slice)
}

func (slice GrantRelationshipRows) Less(i, j int) bool {
	if slice[i]["compare_name"] != slice[j]["compare_name"] {
		return slice[i]["compare_name"] < slice[j]["compare_name"]
	}

	// Only compare the role part of the ACL
	// Not yet sure if this is absolutely necessary
	// (or if we could just compare the entire ACL string)
	relRole1, _ := parseAcl(slice[i]["relationship_acl"])
	relRole2, _ := parseAcl(slice[j]["relationship_acl"])
	if relRole1 != relRole2 {
		return relRole1 < relRole2
	}

	return false
}

func (slice GrantRelationshipRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// GrantRelationshipSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// GrantRelationshipSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type GrantRelationshipSchema struct {
	rows   GrantRelationshipRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *GrantRelationshipSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *GrantRelationshipSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *GrantRelationshipSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *GrantRelationshipSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*GrantRelationshipSchema)
	if !ok {
		fmt.Println("Error!!!, Compare needs a GrantRelationshipSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	if val != 0 {
		return val
	}

	relRole1, _ := parseAcl(c.get("relationship_acl"))
	relRole2, _ := parseAcl(c2.get("relationship_acl"))
	val = misc.CompareStrings(relRole1, relRole2)
	return val

}

// Add prints SQL to add the column
func (c *GrantRelationshipSchema) Add() {
	role, grants := parseGrants(c.get("relationship_acl"))
	fmt.Printf("GRANT %s ON %s TO %s; -- Add\n", strings.Join(grants, ", "), c.get("relationship_name"), role)
}

// Drop prints SQL to drop the column
func (c *GrantRelationshipSchema) Drop() {
	role, grants := parseGrants(c.get("relationship_acl"))
	fmt.Printf("REVOKE %s ON %s FROM %s; -- Drop\n", strings.Join(grants, ", "), c.get("relationship_name"), role)
}

// Change handles the case where the relationship and column match, but the details do not
func (c *GrantRelationshipSchema) Change(obj interface{}) {
	c2, ok := obj.(*GrantRelationshipSchema)
	if !ok {
		fmt.Println("-- Error!!!, Change needs a GrantRelationshipSchema instance", c2)
	}

	role, grants1 := parseGrants(c.get("relationship_acl"))
	_, grants2 := parseGrants(c2.get("relationship_acl"))

	// Find grants in the first db that are not in the second
	// (for this relationship and owner)
	var grantList []string
	for _, g := range grants1 {
		if !misc.ContainsString(grants2, g) {
			grantList = append(grantList, g)
		}
	}
	if len(grantList) > 0 {
		fmt.Printf("GRANT %s ON %s TO %s; -- Change\n", strings.Join(grantList, ", "), c.get("relationship_name"), role)
	}

	// Find grants in the second db that are not in the first
	// (for this relationship and owner)
	var revokeList []string
	for _, g := range grants2 {
		if !misc.ContainsString(grants1, g) {
			revokeList = append(revokeList, g)
		}
	}
	if len(revokeList) > 0 {
		fmt.Printf("REVOKE %s ON %s FROM %s; -- Change\n", strings.Join(revokeList, ", "), c.get("relationship_name"), role)
	}

	//	fmt.Printf("--1 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c.get("relationship_name"), c.get("relationship_acl"), c.get("column_name"), c.get("column_acl"))
	//	fmt.Printf("--2 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c2.get("relationship_name"), c2.get("relationship_acl"), c2.get("column_name"), c2.get("column_acl"))
}

// ==================================
// Functions
// ==================================

// compareGrantRelationships outputs SQL to make the granted permissions match between DBs or schemas
func compareGrantRelationships(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	grantRelationshipSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	grantRelationshipSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(GrantRelationshipRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(GrantRelationshipRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown (to me) reason
	var schema1 Schema = &GrantRelationshipSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &GrantRelationshipSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}
