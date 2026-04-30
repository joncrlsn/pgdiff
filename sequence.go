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
	"strings"
	"text/template"

	"github.com/joncrlsn/misc"
	"github.com/joncrlsn/pgutil"
)

var (
	sequenceSqlTemplate = initSequenceSqlTemplate()
)

// initSequenceSqlTemplate builds the sequence query.
//
// compare_name keys sequences by their *owning column* when one exists
// (pg_depend deptype 'a' = OWNED BY / SERIAL, 'i' = IDENTITY) so that
// renamed-but-otherwise-identical sequences (e.g. foo_id_seq vs foo_id_seq1)
// pair up across the two databases instead of looking like DROP+CREATE.
// Free-floating sequences fall back to schema-qualified name matching.
func initSequenceSqlTemplate() *template.Template {
	sql := `
WITH seq AS (
    SELECT c.oid AS seq_oid, n.nspname AS seq_schema, c.relname AS seq_name
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'S'
      {{if eq $.DbSchema "*" }}
      AND n.nspname NOT LIKE 'pg_%'
      AND n.nspname <> 'information_schema'
      {{else}}
      AND n.nspname = '{{$.DbSchema}}'
      {{end}}
)
SELECT
    s.seq_schema                       AS schema_name,
    s.seq_name                         AS sequence_name,
    ps.seqtypid::regtype::text         AS data_type,
    ps.seqstart::text                  AS start_value,
    ps.seqmin::text                    AS minimum_value,
    ps.seqmax::text                    AS maximum_value,
    ps.seqincrement::text              AS increment,
    CASE WHEN ps.seqcycle THEN 'YES' ELSE 'NO' END AS cycle_option,
    COALESCE(tn.nspname, '')           AS owner_schema,
    COALESCE(t.relname, '')            AS owner_table,
    COALESCE(a.attname, '')            AS owner_column,
    COALESCE(d.deptype::text, '')      AS dep_type,
    CASE
        WHEN t.oid IS NOT NULL THEN
            'O:' || {{if eq $.DbSchema "*" }}tn.nspname || '.' || {{end}}t.relname || '.' || a.attname
        ELSE
            'S:' || {{if eq $.DbSchema "*" }}s.seq_schema || '.' || {{end}}s.seq_name
    END AS compare_name
FROM seq s
JOIN pg_sequence ps ON ps.seqrelid = s.seq_oid
LEFT JOIN pg_depend d
    ON d.objid       = s.seq_oid
   AND d.classid     = 'pg_class'::regclass
   AND d.refclassid  = 'pg_class'::regclass
   AND d.deptype IN ('a', 'i')
LEFT JOIN pg_class     t   ON t.oid = d.refobjid
LEFT JOIN pg_namespace tn  ON tn.oid = t.relnamespace
LEFT JOIN pg_attribute a   ON a.attrelid = d.refobjid AND a.attnum = d.refobjsubid
ORDER BY compare_name;
`
	t := template.New("SequenceSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// SequenceRows definition
// ==================================

// SequenceRows is a sortable slice of string maps
type SequenceRows []map[string]string

func (slice SequenceRows) Len() int { return len(slice) }

func (slice SequenceRows) Less(i, j int) bool {
	return slice[i]["compare_name"] < slice[j]["compare_name"]
}

func (slice SequenceRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// SequenceSchema implements the Schema interface defined in pgdiff.go
type SequenceSchema struct {
	rows   SequenceRows
	rowNum int
	done   bool
}

func (c *SequenceSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

func (c *SequenceSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

func (c *SequenceSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, Compare(obj) needs a SequenceSchema instance", c2)
		return +999
	}
	return misc.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
}

// Add prints SQL to create the sequence on dst.
// IDENTITY-backed sequences are skipped because the COLUMN comparator
// will recreate them implicitly via the column's GENERATED clause.
func (c *SequenceSchema) Add() {
	schema := dbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}
	seqName := c.get("sequence_name")

	if c.get("dep_type") == "i" {
		fmt.Printf("-- skipping CREATE SEQUENCE for IDENTITY-backed %s.%s; the COLUMN diff handles it.\n", schema, seqName)
		return
	}

	fmt.Printf("CREATE SEQUENCE %s AS %s INCREMENT %s MINVALUE %s MAXVALUE %s START %s",
		quoteQualified(schema, seqName),
		c.get("data_type"),
		c.get("increment"),
		c.get("minimum_value"),
		c.get("maximum_value"),
		c.get("start_value"),
	)
	if c.get("cycle_option") == "YES" {
		fmt.Printf(" CYCLE")
	}
	fmt.Printf(";\n")

	// Re-establish OWNED BY when present; this keeps the sequence tied to the column.
	if c.get("owner_table") != "" && c.get("owner_column") != "" {
		ownerSchema := c.get("owner_schema")
		if dbInfo2.DbSchema != "*" {
			ownerSchema = dbInfo2.DbSchema
		}
		fmt.Printf("ALTER SEQUENCE %s OWNED BY %s;\n",
			quoteQualified(schema, seqName),
			quoteQualified(ownerSchema, c.get("owner_table"), c.get("owner_column")))
	}
}

// Drop prints SQL to drop the sequence on dst.
// IDENTITY-backed sequences are skipped (handled by COLUMN diff).
func (c *SequenceSchema) Drop() {
	if c.get("dep_type") == "i" {
		fmt.Printf("-- skipping DROP SEQUENCE for IDENTITY-backed %s.%s; the COLUMN diff handles it.\n",
			c.get("schema_name"), c.get("sequence_name"))
		return
	}
	fmt.Printf("DROP SEQUENCE %s;\n", quoteQualified(c.get("schema_name"), c.get("sequence_name")))
}

// Change emits ALTER statements when paired sequences differ.
//
// If the names differ but the pair was matched by owning column, emit a RENAME
// first — Postgres stores nextval() column defaults as regclass (an OID), so
// the column's default automatically follows the rename. Then any attribute
// drift (start, min/max, increment, cycle) is emitted as a single ALTER.
func (c *SequenceSchema) Change(obj interface{}) {
	c2, ok := obj.(*SequenceSchema)
	if !ok {
		fmt.Println("Error!!!, Change(obj) needs a SequenceSchema instance", c2)
		return
	}

	// IDENTITY sequences are owned by their column's GENERATED clause; let the
	// COLUMN diff manage them rather than emitting ALTER SEQUENCE here.
	if c.get("dep_type") == "i" || c2.get("dep_type") == "i" {
		return
	}

	srcName := c.get("sequence_name")
	dstName := c2.get("sequence_name")
	dstSchema := c2.get("schema_name")

	if srcName != dstName {
		fmt.Printf("ALTER SEQUENCE %s RENAME TO %s; -- matched by owner %s.%s.%s: src=%s dst=%s\n",
			quoteQualified(dstSchema, dstName), quoteIdent(srcName),
			c.get("owner_schema"), c.get("owner_table"), c.get("owner_column"),
			srcName, dstName)
	}

	// After any rename, subsequent ALTERs reference the new (src) name.
	seqName := srcName

	var clauses []string
	if c.get("increment") != c2.get("increment") {
		clauses = append(clauses, "INCREMENT BY "+c.get("increment"))
	}
	if c.get("minimum_value") != c2.get("minimum_value") {
		clauses = append(clauses, "MINVALUE "+c.get("minimum_value"))
	}
	if c.get("maximum_value") != c2.get("maximum_value") {
		clauses = append(clauses, "MAXVALUE "+c.get("maximum_value"))
	}
	if c.get("start_value") != c2.get("start_value") {
		clauses = append(clauses, "START WITH "+c.get("start_value"))
	}
	if c.get("cycle_option") != c2.get("cycle_option") {
		if c.get("cycle_option") == "YES" {
			clauses = append(clauses, "CYCLE")
		} else {
			clauses = append(clauses, "NO CYCLE")
		}
	}
	if len(clauses) > 0 {
		fmt.Printf("ALTER SEQUENCE %s %s;\n", quoteQualified(dstSchema, seqName), strings.Join(clauses, " "))
	}
}

// compareSequences outputs SQL to make the sequences match between DBs or schemas.
func compareSequences(conn1 *sql.DB, conn2 *sql.DB) {
	buf1 := new(bytes.Buffer)
	sequenceSqlTemplate.Execute(buf1, dbInfo1)

	buf2 := new(bytes.Buffer)
	sequenceSqlTemplate.Execute(buf2, dbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(SequenceRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)

	rows2 := make(SequenceRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)

	var schema1 Schema = &SequenceSchema{rows: rows1, rowNum: -1}
	var schema2 Schema = &SequenceSchema{rows: rows2, rowNum: -1}

	doDiff(schema1, schema2)
}
