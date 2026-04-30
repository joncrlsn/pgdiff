//
// Copyright (c) 2026 Hayden Sim.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"sort"
	"text/template"

	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
)

var (
	checkSqlTemplate = initCheckSqlTemplate()
)

// initCheckSqlTemplate builds the CHECK constraint query.
//
// conislocal=true skips constraints inherited from a parent table — those will
// already surface as drift on the parent. contypid=0 excludes domain CHECKs
// (domains are a separate concern).
func initCheckSqlTemplate() *template.Template {
	sql := `
SELECT
    n.nspname  AS schema_name,
    c.relname  AS table_name,
    con.conname AS constraint_name,
    pg_get_constraintdef(con.oid) AS constraint_def,
    {{if eq $.DbSchema "*" }}n.nspname || '.' || {{end}}c.relname || '.' || con.conname AS compare_name
FROM pg_constraint con
JOIN pg_class     c ON c.oid = con.conrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE con.contype = 'c'
  AND con.contypid = 0
  AND con.conislocal = true
  {{if eq $.DbSchema "*" }}
  AND n.nspname NOT LIKE 'pg_%'
  AND n.nspname <> 'information_schema'
  {{else}}
  AND n.nspname = '{{$.DbSchema}}'
  {{end}}
ORDER BY compare_name;
`
	t := template.New("CheckSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// CheckRows is a sortable slice of string maps
type CheckRows []map[string]string

func (slice CheckRows) Len() int { return len(slice) }
func (slice CheckRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}
func (slice CheckRows) Swap(i, j int) { slice[i], slice[j] = slice[j], slice[i] }

// CheckSchema implements the Schema interface defined in pgdiff.go
type CheckSchema struct {
	rows   CheckRows
	rowNum int
	done   bool
}

func (c *CheckSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

func (c *CheckSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

func (c *CheckSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*CheckSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a CheckSchema instance", c2)
		return +999
	}
	return misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
}

func (c *CheckSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n",
		quoteQualified(schema, c.get("table_name")),
		quoteIdent(c.get("constraint_name")),
		c.get("constraint_def"))
}

func (c *CheckSchema) Drop() {
	fmt.Printf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n",
		quoteQualified(c.get("schema_name"), c.get("table_name")),
		quoteIdent(c.get("constraint_name")))
}

// Change handles a definition mismatch. CHECK expressions can't be altered in
// place, so we drop and re-add. The drop targets dst's stored schema; the add
// uses dst's effective schema (which may differ from src's when single-schema
// mode maps a different schema name).
func (c *CheckSchema) Change(obj interface{}) {
	c2, ok := obj.(*CheckSchema)
	if !ok {
		fmt.Println("Error!!!, Change(obj) needs a CheckSchema instance", c2)
		return
	}
	if c.get("constraint_def") == c2.get("constraint_def") {
		return
	}

	dstSchema := c2.get("schema_name")
	addSchema := dbInfo2.DbSchema
	if addSchema == "*" {
		addSchema = dstSchema
	}

	fmt.Printf("-- CHECK constraint drift on %s.%s.%s: src=%q dst=%q\n",
		dstSchema, c2.get("table_name"), c2.get("constraint_name"),
		c.get("constraint_def"), c2.get("constraint_def"))
	fmt.Printf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n",
		quoteQualified(dstSchema, c2.get("table_name")),
		quoteIdent(c2.get("constraint_name")))
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n",
		quoteQualified(addSchema, c.get("table_name")),
		quoteIdent(c.get("constraint_name")),
		c.get("constraint_def"))
}

// isExcludedCheck reports whether a row matches a --exclude-check pattern.
// matched on actual schema.table.constraint regardless of single-schema mode.
func isExcludedCheck(row map[string]string) bool {
	if len(excludeChecks) == 0 {
		return false
	}
	key := row["schema_name"] + "." + row["table_name"] + "." + row["constraint_name"]
	_, skip := excludeChecks[key]
	return skip
}

// compareChecks outputs SQL to make CHECK constraints match between DBs or schemas.
func compareChecks(conn1 *sql.DB, conn2 *sql.DB) {
	buf1 := new(bytes.Buffer)
	checkSqlTemplate.Execute(buf1, dbInfo1)
	buf2 := new(bytes.Buffer)
	checkSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(CheckRows, 0)
	for row := range rowChan1 {
		if isExcludedCheck(row) {
			continue
		}
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(CheckRows, 0)
	for row := range rowChan2 {
		if isExcludedCheck(row) {
			continue
		}
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	var schema1 Schema = &CheckSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &CheckSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}
