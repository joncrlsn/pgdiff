//
// Copyright (c) 2026 Hayden Sim.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
)

var (
	enumSqlTemplate = initEnumSqlTemplate()
)

// initEnumSqlTemplate builds the enum-type query.
//
// Values come back as a JSON array (in enumsortorder) so we can compare both
// membership and ordering accurately in Go.
func initEnumSqlTemplate() *template.Template {
	sql := `
SELECT
    n.nspname AS schema_name,
    t.typname AS enum_name,
    COALESCE(
        (SELECT to_json(array_agg(e.enumlabel ORDER BY e.enumsortorder))::text
         FROM pg_enum e WHERE e.enumtypid = t.oid),
        '[]'
    ) AS enum_values_json,
    {{if eq $.DbSchema "*" }}n.nspname || '.' || {{end}}t.typname AS compare_name
FROM pg_type t
JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'e'
  {{if eq $.DbSchema "*" }}
  AND n.nspname NOT LIKE 'pg_%'
  AND n.nspname <> 'information_schema'
  {{else}}
  AND n.nspname = '{{$.DbSchema}}'
  {{end}}
ORDER BY compare_name;
`
	t := template.New("EnumSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// EnumRows is a sortable slice of string maps
type EnumRows []map[string]string

func (slice EnumRows) Len() int { return len(slice) }
func (slice EnumRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}
func (slice EnumRows) Swap(i, j int) { slice[i], slice[j] = slice[j], slice[i] }

// EnumSchema implements the Schema interface defined in pgdiff.go
type EnumSchema struct {
	rows   EnumRows
	rowNum int
	done   bool
}

func (c *EnumSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

func (c *EnumSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

func (c *EnumSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*EnumSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs an EnumSchema instance", c2)
		return +999
	}
	return misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
}

// values parses the JSON-encoded enum values column into an ordered slice.
func (c *EnumSchema) values() []string {
	raw := c.get("enum_values_json")
	if raw == "" || raw == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		fmt.Printf("-- WARNING: failed to parse enum values for %s: %v\n", c.get("compare_name"), err)
		return nil
	}
	return out
}

func (c *EnumSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}
	vals := c.values()
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = pgQuoteLiteral(v)
	}
	fmt.Printf("CREATE TYPE %s AS ENUM (%s);\n",
		quoteQualified(schema, c.get("enum_name")), strings.Join(quoted, ", "))
}

func (c *EnumSchema) Drop() {
	fmt.Printf("DROP TYPE %s;\n", quoteQualified(c.get("schema_name"), c.get("enum_name")))
}

// Change emits ADD VALUE for new entries on the source side, with BEFORE/AFTER
// positioning to preserve ordering. Removed entries and reorderings of values
// present on both sides cannot be expressed as ALTER — those require dropping
// and recreating the type (which fails if any column references it), so we
// emit a warning and let the operator decide.
func (c *EnumSchema) Change(obj interface{}) {
	c2, ok := obj.(*EnumSchema)
	if !ok {
		fmt.Println("Error!!!, Change(obj) needs an EnumSchema instance", c2)
		return
	}

	srcVals := c.values()
	dstVals := c2.values()

	dstSet := make(map[string]struct{}, len(dstVals))
	for _, v := range dstVals {
		dstSet[v] = struct{}{}
	}
	srcSet := make(map[string]struct{}, len(srcVals))
	for _, v := range srcVals {
		srcSet[v] = struct{}{}
	}

	enumName := c.get("enum_name")
	dstSchema := c2.get("schema_name")

	// 1. Add values present in src but missing in dst, with BEFORE/AFTER positioning.
	for i, v := range srcVals {
		if _, exists := dstSet[v]; exists {
			continue
		}
		clause := positionClause(srcVals, i, dstSet)
		if clause == "" {
			fmt.Printf("ALTER TYPE %s ADD VALUE IF NOT EXISTS %s;\n",
				quoteQualified(dstSchema, enumName), pgQuoteLiteral(v))
		} else {
			fmt.Printf("ALTER TYPE %s ADD VALUE IF NOT EXISTS %s %s;\n",
				quoteQualified(dstSchema, enumName), pgQuoteLiteral(v), clause)
		}
	}

	// 2. Warn about values in dst that no longer exist in src — can't drop in place.
	var removed []string
	for _, v := range dstVals {
		if _, exists := srcSet[v]; !exists {
			removed = append(removed, v)
		}
	}
	if len(removed) > 0 {
		fmt.Printf("-- WARNING: enum %s.%s has values on dst that are missing from src: %s\n",
			dstSchema, enumName, strings.Join(removed, ", "))
		fmt.Println("--          Postgres does not support removing enum values; recreate the type if you really need to drop them.")
	}

	// 3. Warn about ordering drift among the values that exist on both sides.
	var commonSrc, commonDst []string
	for _, v := range srcVals {
		if _, exists := dstSet[v]; exists {
			commonSrc = append(commonSrc, v)
		}
	}
	for _, v := range dstVals {
		if _, exists := srcSet[v]; exists {
			commonDst = append(commonDst, v)
		}
	}
	if !stringSlicesEqual(commonSrc, commonDst) {
		fmt.Printf("-- WARNING: enum %s.%s has reordered values relative to src — Postgres can't reorder existing labels in place.\n",
			dstSchema, enumName)
		fmt.Printf("--          src order: %s\n", strings.Join(srcVals, ", "))
		fmt.Printf("--          dst order: %s\n", strings.Join(dstVals, ", "))
	}
}

// positionClause computes a BEFORE/AFTER clause to preserve src ordering when
// inserting srcVals[i] into dst. It looks for the nearest neighbor that already
// exists on dst — preferring AFTER the previous, falling back to BEFORE the
// next. Returns "" if no positioned neighbor exists (value will be appended).
func positionClause(srcVals []string, i int, dstSet map[string]struct{}) string {
	for j := i - 1; j >= 0; j-- {
		if _, exists := dstSet[srcVals[j]]; exists {
			return "AFTER " + pgQuoteLiteral(srcVals[j])
		}
	}
	for j := i + 1; j < len(srcVals); j++ {
		if _, exists := dstSet[srcVals[j]]; exists {
			return "BEFORE " + pgQuoteLiteral(srcVals[j])
		}
	}
	return ""
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// pgQuoteLiteral wraps a string as a Postgres single-quoted string literal,
// escaping embedded single quotes by doubling.
func pgQuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// compareEnums outputs SQL to make enum types match between DBs or schemas.
func compareEnums(conn1 *sql.DB, conn2 *sql.DB) {
	buf1 := new(bytes.Buffer)
	enumSqlTemplate.Execute(buf1, dbInfo1)
	buf2 := new(bytes.Buffer)
	enumSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(EnumRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(EnumRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	var schema1 Schema = &EnumSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &EnumSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}
