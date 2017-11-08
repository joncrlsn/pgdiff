//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"text/template"
	"bytes"
	"database/sql"
	"fmt"
	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
	"sort"
	"strings"
)

var (
	grantAttributeSqlTemplate = initGrantAttributeSqlTemplate()
)

// Initializes the Sql template
func initGrantAttributeSqlTemplate() *template.Template {
	sql := `
-- Attribute/Column ACL only
SELECT
  n.nspname AS schema
  , CASE c.relkind
    WHEN 'r' THEN 'TABLE'
    WHEN 'v' THEN 'VIEW'
    WHEN 'f' THEN 'FOREIGN TABLE'
    END as type
  , c.relname AS relationship_name
  , a.attname AS attribute_name
  , a.attacl AS attribute_acl
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
INNER JOIN (SELECT attname, unnest(attacl) AS attacl, attrelid
           FROM pg_catalog.pg_attribute
           WHERE NOT attisdropped AND attacl IS NOT NULL)
      AS a ON (a.attrelid = c.oid)
WHERE c.relkind IN ('r', 'v', 'f')
AND n.nspname !~ '^pg_' 
AND pg_catalog.pg_table_is_visible(c.oid)
ORDER BY n.nspname, c.relname, a.attname;
`

	//sql := `
	//SELECT n.nspname                 AS schema_name
	//, {{if eq $.DbSchema "*" }}n.nspname || '.' || {{end}}p.proname AS compare_name
	//, p.proname                  AS function_name
	//, p.oid::regprocedure        AS fancy
	//, t.typname                  AS return_type
	//, pg_get_functiondef(p.oid)  AS definition
	//FROM pg_proc AS p
	//JOIN pg_type t ON (p.prorettype = t.oid)
	//JOIN pg_namespace n ON (n.oid = p.pronamespace)
	//JOIN pg_language l ON (p.prolang = l.oid AND l.lanname IN ('c','plpgsql', 'sql'))
	//WHERE true
	//{{if eq $.DbSchema "*" }}
	//AND n.nspname NOT LIKE 'pg_%'
	//AND n.nspname <> 'information_schema'
	//{{else}}
	//AND n.nspname = '{{$.DbSchema}}'
	//{{end}};
	//`
	t := template.New("GrantAttributeSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// GrantAttributeRows definition
// ==================================

// GrantAttributeRows is a sortable slice of string maps
type GrantAttributeRows []map[string]string

func (slice GrantAttributeRows) Len() int {
	return len(slice)
}

func (slice GrantAttributeRows) Less(i, j int) bool {
	if slice[i]["schema"] != slice[j]["schema"] {
		return slice[i]["schema"] < slice[j]["schema"]
	}
	if slice[i]["relationship_name"] != slice[j]["relationship_name"] {
		return slice[i]["relationship_name"] < slice[j]["relationship_name"]
	}
	if slice[i]["attribute_name"] != slice[j]["attribute_name"] {
		return slice[i]["attribute_name"] < slice[j]["attribute_name"]
	}

	// Only compare the role part of the ACL
	// Not yet sure if this is absolutely necessary
	// (or if we could just compare the entire ACL string)
	role1, _ := parseAcl(slice[i]["attribute_acl"])
	role2, _ := parseAcl(slice[j]["attribute_acl"])
	if role1 != role2 {
		return role1 < role2
	}

	return false
}

func (slice GrantAttributeRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// GrantAttributeSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// GrantAttributeSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type GrantAttributeSchema struct {
	rows   GrantAttributeRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *GrantAttributeSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *GrantAttributeSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *GrantAttributeSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *GrantAttributeSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*GrantAttributeSchema)
	if !ok {
		fmt.Println("Error!!!, Compare needs a GrantAttributeSchema instance", c2)
		return +999
	}

	val := misc.CompareStrings(c.get("schema"), c2.get("schema"))
	if val != 0 {
		return val
	}

	val = misc.CompareStrings(c.get("relationship_name"), c2.get("relationship_name"))
	if val != 0 {
		return val
	}

	val = misc.CompareStrings(c.get("attribute_name"), c2.get("attribute_name"))
	if val != 0 {
		return val
	}

	role1, _ := parseAcl(c.get("attribute_acl"))
	role2, _ := parseAcl(c2.get("attribute_acl"))
	val = misc.CompareStrings(role1, role2)
	return val
}

// Add prints SQL to add the column
func (c *GrantAttributeSchema) Add() {
	role, grants := parseGrants(c.get("attribute_acl"))
	fmt.Printf("GRANT %s (%s) ON %s TO %s; -- Add\n", strings.Join(grants, ", "), c.get("attribute_name"), c.get("relationship_name"), role)
}

// Drop prints SQL to drop the column
func (c *GrantAttributeSchema) Drop() {
	role, grants := parseGrants(c.get("attribute_acl"))
	fmt.Printf("REVOKE %s (%s) ON %s FROM %s; -- Drop\n", strings.Join(grants, ", "), c.get("attribute_name"), c.get("relationship_name"), role)
}

// Change handles the case where the relationship and column match, but the details do not
func (c *GrantAttributeSchema) Change(obj interface{}) {
	c2, ok := obj.(*GrantAttributeSchema)
	if !ok {
		fmt.Println("-- Error!!!, Change needs a GrantAttributeSchema instance", c2)
	}

	role, grants1 := parseGrants(c.get("attribute_acl"))
	_, grants2 := parseGrants(c2.get("attribute_acl"))

	// Find grants in the first db that are not in the second
	// (for this relationship and owner)
	var grantList []string
	for _, g := range grants1 {
		if !misc.ContainsString(grants2, g) {
			grantList = append(grantList, g)
		}
	}
	if len(grantList) > 0 {
		fmt.Printf("GRANT %s (%s) ON %s TO %s; -- Change\n", strings.Join(grantList, ", "), c.get("attribute_name"), c.get("relationship_name"), role)
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
		fmt.Printf("REVOKE %s (%s) ON %s FROM %s; -- Change\n", strings.Join(grantList, ", "), c.get("attribute_name"), c.get("relationship_name"), role)
	}

	//	fmt.Printf("--1 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c.get("attribute_name"), c.get("attribute_acl"), c.get("attribute_name"), c.get("attribute_acl"))
	//	fmt.Printf("--2 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c2.get("attribute_name"), c2.get("attribute_acl"), c2.get("attribute_name"), c2.get("attribute_acl"))
}

// ==================================
// Functions
// ==================================

// compareGrantAttributes outputs SQL to make the granted permissions match between DBs or schemas
func compareGrantAttributes(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	grantAttributeSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	grantAttributeSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(GrantAttributeRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(GrantAttributeRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &GrantAttributeSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &GrantAttributeSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}
