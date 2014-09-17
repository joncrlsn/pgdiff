package main

import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"

// PrimaryKeySchema holds a channel streaming foreign key data from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// PrimaryKeySchema implements the Schema interface defined in pgdiff.go
type PrimaryKeySchema struct {
	channel chan map[string]string
	row     map[string]string
	done    bool
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *PrimaryKeySchema) NextRow() bool {
	c.row = <-c.channel
	if len(c.row) == 0 {
		c.done = true
	}
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *PrimaryKeySchema) Compare(obj interface{}) int {
	c2, ok := obj.(*PrimaryKeySchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a PrimaryKeySchema instance", c2)
		return +999
	}
	val := _compareString(c.row["table_name"], c2.row["table_name"])
	if val != 0 {
		return val
	}

	return _compareString(c.row["constraint_def"], c2.row["constraint_def"])
}

// Add returns SQL to add the primary key
func (c PrimaryKeySchema) Add() {
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n", c.row["table_name"], c.row["constraint_name"], c.row["constraint_def"])
}

// Drop returns SQL to drop the foreign key
func (c PrimaryKeySchema) Drop() {
	fmt.Printf("ALTER TABLE %s DROP CONSTRAINT %s; -- %s\n", c.row["table_name"], c.row["constraint_name"], c.row["constraint_def"])
}

// Change handles the case where the table name matches, but the details do not
func (c PrimaryKeySchema) Change(obj interface{}) {
	c2, ok := obj.(*PrimaryKeySchema)
	if !ok {
		fmt.Println("Error!!!, change needs a PrimaryKeySchema instance", c2)
	}
	// There is no "changing" a primary key.  It either gets created or dropped (or left as-is).
}

/*
 * Compare the primary keys in the two databases.  We do not recreate primary keys if just the name is different.
 */
func comparePrimaryKeys(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT c.conname AS constraint_name
	, c.contype AS constraint_type
	, cl.relname AS table_name
	, pg_catalog.pg_get_constraintdef(c.oid, true) as constraint_def
FROM pg_catalog.pg_constraint c
INNER JOIN pg_class AS cl ON (c.conrelid = cl.oid)
WHERE c.contype = 'p'
ORDER BY cl.relname::varchar, pg_catalog.pg_get_constraintdef(c.oid, true) COLLATE "C" ASC;
`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some unknown reason
	var schema1 Schema = &PrimaryKeySchema{channel: rowChan1}
	var schema2 Schema = &PrimaryKeySchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
}
