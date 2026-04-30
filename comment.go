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
	commentSqlTemplate = initCommentSqlTemplate()
)

// initCommentSqlTemplate builds the comment query.
//
// Comments live in pg_description, keyed by (classoid, objoid, objsubid).
// We resolve each entry into the keyword + identifier the COMMENT statement
// expects (e.g. "TABLE public.users", "COLUMN public.users.email",
// "FUNCTION public.foo(integer, text)"), so the diff output is just a
// straight `COMMENT ON <target> IS '...'`. The compare_name namespaces by
// kind so e.g. a TABLE and a TYPE with the same name don't collide.
//
// Object kinds covered: tables, views, matviews, foreign tables, indexes,
// sequences, columns (of any of those), functions/procedures, types
// (incl. domains/enums/composites), schemas. Less common kinds (operators,
// casts, FDWs, etc.) are intentionally out of scope.
func initCommentSqlTemplate() *template.Template {
	sql := `
WITH
class_comments AS (
    SELECT
        d.objoid, d.classoid, d.objsubid, d.description,
        n.nspname AS schema_name,
        c.relname AS object_name,
        c.relkind,
        a.attname AS column_name
    FROM pg_description d
    JOIN pg_class      c ON c.oid = d.objoid AND d.classoid = 'pg_class'::regclass
    JOIN pg_namespace  n ON n.oid = c.relnamespace
    LEFT JOIN pg_attribute a
        ON a.attrelid = c.oid AND a.attnum = d.objsubid AND d.objsubid > 0
    WHERE c.relkind IN ('r','v','m','f','p','i','S')
      {{if eq $.DbSchema "*" }}
      AND n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      {{else}}
      AND n.nspname = '{{$.DbSchema}}'
      {{end}}
),
proc_comments AS (
    SELECT
        d.objoid, d.classoid, 0 AS objsubid, d.description,
        n.nspname AS schema_name,
        p.proname AS object_name,
        pg_get_function_identity_arguments(p.oid) AS proc_args
    FROM pg_description d
    JOIN pg_proc       p ON p.oid = d.objoid AND d.classoid = 'pg_proc'::regclass
    JOIN pg_namespace  n ON n.oid = p.pronamespace
    WHERE 1=1
      {{if eq $.DbSchema "*" }}
      AND n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      {{else}}
      AND n.nspname = '{{$.DbSchema}}'
      {{end}}
),
type_comments AS (
    SELECT
        d.objoid, d.classoid, 0 AS objsubid, d.description,
        n.nspname AS schema_name,
        t.typname AS object_name
    FROM pg_description d
    JOIN pg_type       t ON t.oid = d.objoid AND d.classoid = 'pg_type'::regclass
    JOIN pg_namespace  n ON n.oid = t.typnamespace
    WHERE t.typrelid = 0  -- skip composites that mirror a relation; comment lives on the relation
      {{if eq $.DbSchema "*" }}
      AND n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      {{else}}
      AND n.nspname = '{{$.DbSchema}}'
      {{end}}
),
schema_comments AS (
    SELECT
        d.objoid, d.classoid, 0 AS objsubid, d.description,
        nm.nspname AS schema_name,
        nm.nspname AS object_name
    FROM pg_description d
    JOIN pg_namespace nm ON nm.oid = d.objoid AND d.classoid = 'pg_namespace'::regclass
    WHERE 1=1
      {{if eq $.DbSchema "*" }}
      AND nm.nspname NOT LIKE 'pg_%'
      AND nm.nspname <> 'information_schema'
      {{else}}
      AND nm.nspname = '{{$.DbSchema}}'
      {{end}}
)
SELECT kind, schema_name, object_name, column_name, proc_args, description, compare_name FROM (
    SELECT
        CASE
            WHEN cc.objsubid > 0 THEN 'COLUMN'
            WHEN cc.relkind IN ('r','p','f') THEN 'TABLE'
            WHEN cc.relkind = 'v' THEN 'VIEW'
            WHEN cc.relkind = 'm' THEN 'MATERIALIZED VIEW'
            WHEN cc.relkind = 'i' THEN 'INDEX'
            WHEN cc.relkind = 'S' THEN 'SEQUENCE'
        END AS kind,
        cc.schema_name,
        cc.object_name,
        COALESCE(cc.column_name, '') AS column_name,
        ''::text AS proc_args,
        cc.description,
        CASE
            WHEN cc.objsubid > 0 THEN
                'COL:' || {{if eq $.DbSchema "*" }}cc.schema_name || '.' || {{end}}cc.object_name || '.' || cc.column_name
            WHEN cc.relkind IN ('r','p','f') THEN
                'TBL:' || {{if eq $.DbSchema "*" }}cc.schema_name || '.' || {{end}}cc.object_name
            WHEN cc.relkind = 'v' THEN
                'VW:'  || {{if eq $.DbSchema "*" }}cc.schema_name || '.' || {{end}}cc.object_name
            WHEN cc.relkind = 'm' THEN
                'MV:'  || {{if eq $.DbSchema "*" }}cc.schema_name || '.' || {{end}}cc.object_name
            WHEN cc.relkind = 'i' THEN
                'IDX:' || {{if eq $.DbSchema "*" }}cc.schema_name || '.' || {{end}}cc.object_name
            WHEN cc.relkind = 'S' THEN
                'SEQ:' || {{if eq $.DbSchema "*" }}cc.schema_name || '.' || {{end}}cc.object_name
        END AS compare_name
    FROM class_comments cc
    UNION ALL
    SELECT
        'FUNCTION' AS kind,
        schema_name,
        object_name,
        '' AS column_name,
        COALESCE(proc_args, '') AS proc_args,
        description,
        'FN:' || {{if eq $.DbSchema "*" }}schema_name || '.' || {{end}}object_name || '(' || COALESCE(proc_args, '') || ')' AS compare_name
    FROM proc_comments
    UNION ALL
    SELECT
        'TYPE' AS kind,
        schema_name,
        object_name,
        '' AS column_name,
        '' AS proc_args,
        description,
        'TYP:' || {{if eq $.DbSchema "*" }}schema_name || '.' || {{end}}object_name AS compare_name
    FROM type_comments
    UNION ALL
    SELECT
        'SCHEMA' AS kind,
        ''::text AS schema_name,
        object_name,
        '' AS column_name,
        '' AS proc_args,
        description,
        'SCH:' || object_name AS compare_name
    FROM schema_comments
) all_comments
WHERE kind IS NOT NULL
ORDER BY compare_name;
`
	t := template.New("CommentSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// CommentRows is a sortable slice of string maps
type CommentRows []map[string]string

func (slice CommentRows) Len() int { return len(slice) }
func (slice CommentRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}
func (slice CommentRows) Swap(i, j int) { slice[i], slice[j] = slice[j], slice[i] }

// CommentSchema implements the Schema interface defined in pgdiff.go
type CommentSchema struct {
	rows   CommentRows
	rowNum int
	done   bool
}

func (c *CommentSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

func (c *CommentSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

func (c *CommentSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*CommentSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a CommentSchema instance", c2)
		return +999
	}
	return misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
}

// targetIdent assembles the quoted identifier the COMMENT statement needs,
// shaped per object kind. Function args are pre-formatted by Postgres
// (pg_get_function_identity_arguments) and are types not identifiers, so they
// pass through unquoted.
func (c *CommentSchema) targetIdent() string {
	kind := c.get("kind")
	schema := c.get("schema_name")
	object := c.get("object_name")
	switch kind {
	case "SCHEMA":
		return quoteIdent(object)
	case "COLUMN":
		return quoteQualified(schema, object, c.get("column_name"))
	case "FUNCTION":
		return quoteQualified(schema, object) + "(" + c.get("proc_args") + ")"
	default:
		return quoteQualified(schema, object)
	}
}

// Add: src has a comment that dst lacks → set it.
func (c *CommentSchema) Add() {
	fmt.Printf("COMMENT ON %s %s IS %s;\n",
		c.get("kind"), c.targetIdent(), pgQuoteLiteral(c.get("description")))
}

// Drop: dst has a comment that src lacks → clear it (`IS NULL`).
func (c *CommentSchema) Drop() {
	fmt.Printf("COMMENT ON %s %s IS NULL;\n", c.get("kind"), c.targetIdent())
}

// Change: both sides have a comment but the text differs → overwrite with src's.
func (c *CommentSchema) Change(obj interface{}) {
	c2, ok := obj.(*CommentSchema)
	if !ok {
		fmt.Println("Error!!!, Change(obj) needs a CommentSchema instance", c2)
		return
	}
	if c.get("description") == c2.get("description") {
		return
	}
	fmt.Printf("COMMENT ON %s %s IS %s;\n",
		c.get("kind"), c.targetIdent(), pgQuoteLiteral(c.get("description")))
}

// compareComments outputs SQL to make object comments match between DBs or schemas.
func compareComments(conn1 *sql.DB, conn2 *sql.DB) {
	buf1 := new(bytes.Buffer)
	commentSqlTemplate.Execute(buf1, dbInfo1)
	buf2 := new(bytes.Buffer)
	commentSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(CommentRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(CommentRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	var schema1 Schema = &CommentSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &CommentSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}

