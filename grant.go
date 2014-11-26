package main

import "sort"
import "os"
import "fmt"
import "strings"
import "regexp"
import "database/sql"
import "github.com/joncrlsn/pgutil"

var aclRegex = regexp.MustCompile(`([a-zA-Z0-9]+)*=([rwadDxtXUCcT]+)/([a-zA-Z0-9]+)$`)

var permMap map[string]string = map[string]string{
	"a": "INSERT",
	"r": "SELECT",
	"w": "UPDATE",
	"d": "DELETE",
	"D": "TRUNCATE",
	"x": "REFERENCES",
	"t": "TRIGGER",
	"X": "EXECUTE",
	"U": "USAGE",
	"C": "CREATE",
	"c": "CONNECT",
	"T": "TEMPORARY",
}

// ==================================
// GrantRows definition (an array of string maps)
// ==================================
type GrantRows []map[string]string

func (slice GrantRows) Len() int {
	return len(slice)
}

func (slice GrantRows) Less(i, j int) bool {
	if slice[i]["schema"] != slice[j]["schema"] {
		return slice[i]["schema"] < slice[j]["schema"]
	}
	if slice[i]["relationship_name"] != slice[j]["relationship_name"] {
		return slice[i]["relationship_name"] < slice[j]["relationship_name"]
	}
	if slice[i]["column_name"] != slice[j]["column_name"] {
		return slice[i]["column_name"] < slice[j]["column_name"]
	}
	if slice[i]["relationship_acl"] != slice[j]["relationship_acl"] {
		return slice[i]["relationship_acl"] < slice[j]["relationship_acl"]
	}
	if slice[i]["column_acl"] != slice[j]["column_acl"] {
		return slice[i]["column_acl"] < slice[j]["column_acl"]
	}
	return false
}

func (slice GrantRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// GrantSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// GrantSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type GrantSchema struct {
	rows   GrantRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *GrantSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *GrantSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *GrantSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *GrantSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*GrantSchema)
	if !ok {
		fmt.Println("Error!!!, Compare needs a GrantSchema instance", c2)
		return +999
	}

	val := _compareString(c.get("schema"), c2.get("schema"))
	if val != 0 {
		return val
	}

	val = _compareString(c.get("relationship_name"), c2.get("relationship_name"))
	if val != 0 {
		return val
	}

	val = _compareString(c.get("column_name"), c2.get("column_name"))
	return val
}

// Add prints SQL to add the column
func (c *GrantSchema) Add() {
	fmt.Println("--Add")

	acls := parseGrants(c.get("relationship_acl"))
	for _, acl := range acls {
		fmt.Printf("GRANT %s ON %s TO %s;\n", strings.Join(acl.grants, ", "), c.get("relationship_name"), acl.role)
	}

	acls = parseGrants(c.get("column_acl"))
	for _, acl := range acls {
		fmt.Printf("GRANT %s (%s) ON %s TO %s;\n", strings.Join(acl.grants, ", "), c.get("column_name"), c.get("relationship_name"), acl.role)
	}
}

// Drop prints SQL to drop the column
func (c *GrantSchema) Drop() {
	fmt.Println("--Drop")

	acls := parseGrants(c.get("relationship_acl"))
	for _, acl := range acls {
		fmt.Printf("REVOKE %s ON %s TO %s;\n", strings.Join(acl.grants, ", "), c.get("relationship_name"), acl.role)
	}

	acls = parseGrants(c.get("column_acl"))
	for _, acl := range acls {
		fmt.Printf("REVOKE %s (%s) ON %s TO %s;\n", strings.Join(acl.grants, ", "), c.get("column_name"), c.get("relationship_name"), acl.role)
	}
}

// Change handles the case where the relationship and column match, but the details do not
func (c *GrantSchema) Change(obj interface{}) {
	c2, ok := obj.(*GrantSchema)
	if !ok {
		fmt.Println("-- Error!!!, change needs a GrantSchema instance", c2)
	}

	{
		acls1 := parseGrants(c.get("relationship_acl"))
		acls2 := parseGrants(c2.get("relationship_acl"))
		_diffGrants(acls1, acls2, c.get("relationship_name"), c.get("column_name"))
	}

	{
		//	if c.get("column_acl") != "null" && len(c.get("column_acl")) > 0 {
		acls1 := parseGrants(c.get("column_acl"))
		acls2 := parseGrants(c2.get("column_acl"))
		_diffGrants(acls1, acls2, c.get("relationship_name"), c.get("column_name"))
	}

	fmt.Println("--Change")
	fmt.Printf("--1 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c.get("relationship_name"), c.get("relationship_acl"), c.get("column_name"), c.get("column_acl"))
	fmt.Printf("--2 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c2.get("relationship_name"), c2.get("relationship_acl"), c2.get("column_name"), c2.get("column_acl"))
}

// ==================================
// Functions
// ==================================

/*
 * Compare the columns in the two databases
 */
func compareGrants(conn1 *sql.DB, conn2 *sql.DB) {
    fmt.Println(" Grant is broken right now. Please come back later! :-)")
    os.Exit(0)

	sql := `
SELECT
  n.nspname AS schema
  , CASE c.relkind
    WHEN 'r' THEN 'TABLE'
        WHEN 'v' THEN 'VIEW'
        WHEN 'S' THEN 'SEQUENCE'
        WHEN 'f' THEN 'FOREIGN TABLE'
    END as type
  , c.relname AS relationship_name
--  , pg_catalog.array_to_string(c.relacl, E'\n') AS relationship_acl
--  , pg_catalog.array_to_string(a.attacl, E'\n') AS column_acl
  , unnest(c.relacl) AS relationship_acl
  , a.attname AS column_name
  , a.attacl AS column_acl
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
LEFT JOIN (SELECT attname, unnest(attacl) AS attacl, attrelid, attisdropped 
           FROM pg_catalog.pg_attribute 
           WHERE NOT attisdropped AND attacl IS NOT NULL) 
      AS a ON (a.attrelid = c.oid)
WHERE c.relkind IN ('r', 'v', 'S', 'f')
  AND n.nspname !~ '^pg_' AND pg_catalog.pg_table_is_visible(c.oid)
ORDER BY n.nspname, c.relname, a.attname;
`
//oldSql := `SELECT
//  n.nspname AS schema
//  , CASE c.relkind
//    	WHEN 'r' THEN 'TABLE'
//		WHEN 'v' THEN 'VIEW'
//		WHEN 'S' THEN 'SEQUENCE'
//		WHEN 'f' THEN 'FOREIGN TABLE'
//	END as type
//  , c.relname AS relationship_name
//  , pg_catalog.array_to_string(c.relacl, E'\n') AS relationship_acl
//  , a.attname AS column_name
//  , pg_catalog.array_to_string(a.attacl, E'\n') AS column_acl
//FROM pg_catalog.pg_class c
//LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
//LEFT JOIN pg_catalog.pg_attribute a ON (a.attrelid = c.oid AND NOT a.attisdropped AND a.attacl IS NOT NULL)
//WHERE c.relkind IN ('r', 'v', 'S', 'f')
//  AND n.nspname !~ '^pg_' AND pg_catalog.pg_table_is_visible(c.oid);
//`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	rows1 := make(GrantRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(GrantRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 Schema = &GrantSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &GrantSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}

// ==================================
// Private functions and structures
// ==================================

func _diffGrants(acls1 RoleAcls, acls2 RoleAcls, table string, column string) {
	//fmt.Printf("GRANT %s (%s) ON %s TO %s; \n", strings.Join(perms, ", "), c.get("column_name"), c.get("relationship_name"), roleName)
	ix1, ix2 := 0, 0
	more1 := ix1 < len(acls1)
	more2 := ix2 < len(acls2)
	var acl1 RoleAcl
	var acl2 RoleAcl
	if more1 {
		acl1 = acls1[ix1]
	}
	if more2 {
		acl2 = acls2[ix2]
	}
	for more1 || more2 {
		if acl1.role == acl2.role {
			//_diffRole(acl1.grants, acl2.grants, acl1.role, table, column)
			ix1 += 1
			ix2 += 1
			more1 := ix1 < len(acls1)
			more2 := ix2 < len(acls2)
			if more1 {
				acl1 = acls1[ix1]
			}
			if more2 {
				acl2 = acls2[ix2]
			}
		} else if acl1.role < acl2.role {
		} else if acl1.role > acl2.role {
		}
		//	compareVal := db1.Compare(db2)
		//	if compareVal == 0 {
		//		// table and column match, look for non-identifying changes
		//		db1.Change(db2)
		//		more1 = db1.NextRow()
		//		more2 = db2.NextRow()
		//	} else if compareVal < 0 {
		//		// db2 is missing a value that db1 has
		//		if more1 {
		//			db1.Add()
		//			more1 = db1.NextRow()
		//		} else {
		//			// db1 is at the end
		//			db2.Drop()
		//			more2 = db2.NextRow()
		//		}
		//	} else if compareVal > 0 {
		//		// db2 has an extra column that we don't want
		//		if more2 {
		//			db2.Drop()
		//			more2 = db2.NextRow()
		//		} else {
		//			// db2 is at the end
		//			db1.Add()
		//			more1 = db1.NextRow()
		//		}
		//	}
	}
}

//
// RoleAcls (a sortable slice of RoleAcl instances)
//
type RoleAcls []RoleAcl

func (slice RoleAcls) Len() int {
	return len(slice)
}

func (slice RoleAcls) Less(i, j int) bool {
	return slice[i].role < slice[j].role
}

func (slice RoleAcls) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (slice RoleAcls) get(role string) RoleAcl {
	for _, roleAcl := range slice {
		if roleAcl.role == role {
			return roleAcl
		}
	}
	return RoleAcl{role: "", grants: []string{}}
}

//
// RoleAcl
//
type RoleAcl struct {
	role   string
	grants []string
}

// parseGrants breaks up a set of ACL lines and parses them into a slice of permission strings per line.
func parseGrants(acl string) (roleAcls RoleAcls) {
	lines := strings.Split(acl, "\n")
	roleAcls = make(RoleAcls, 0)
	for _, line := range lines {
		roleName, perms := _parseGrants(line)
		roleAcl := RoleAcl{role: roleName, grants: perms}
		roleAcls = append(roleAcls, roleAcl)
	}
	sort.Sort(roleAcls)
	return
}

/*
_parseGrants converts an ACL line into a slice of permission strings

Example of an ACL: c42ro=rwa/c42  (we want to separate out the "rwa" part)

rolename=xxxx -- privileges granted to a role
        =xxxx -- privileges granted to PUBLIC
            r -- SELECT ("read")
            w -- UPDATE ("write")
            a -- INSERT ("append")
            d -- DELETE
            D -- TRUNCATE
            x -- REFERENCES
            t -- TRIGGER
            X -- EXECUTE
            U -- USAGE
            C -- CREATE
            c -- CONNECT
            T -- TEMPORARY
      arwdDxt -- ALL PRIVILEGES (for tables, varies for other objects)
            * -- grant option for preceding privilege
        /yyyy -- role that granted this privilege
*/
func _parseGrants(acl string) (string, sort.StringSlice) {
	matches := aclRegex.FindStringSubmatch(acl)
	role := matches[1]
	perms := matches[2]
	permWords := make(sort.StringSlice, 0)
	for _, c := range strings.Split(perms, "") {
		permWord := permMap[c]
		if len(permWord) > 0 {
			permWords = append(permWords, permWord)
		}
	}
	permWords.Sort()
	return role, permWords
}
