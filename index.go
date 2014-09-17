package main

import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"

// IndexSchema holds a channel that streams index metadata from one of the databases.
// It also holds a reference to the current row of data we're viewing.
//
// IndexSchema implements the Schema interface defined in pgdiff.go
type IndexSchema struct {
	channel chan map[string]string
	row     map[string]string
	done    bool
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *IndexSchema) NextRow() bool {
	c.row = <-c.channel
	if len(c.row) == 0 {
		c.done = true
	}
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *IndexSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*IndexSchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a IndexSchema instance", c2)
		return +999
	}

	//fmt.Printf("Comparing %s with %s", c.row["table_name"], c2.row["table_name"])
	val := _compareString(c.row["table_name"], c2.row["table_name"])
	if val != 0 {
		return val
	}

	val = _compareString(c.row["index_name"], c2.row["index_name"])
	return val
}

// Add generates SQL to add the index
func (c IndexSchema) Add() {
	uniqueStr := ""
	if c.row["unique"] == "true" {
		uniqueStr = "UNIQUE "
	}
	fmt.Printf("CREATE %sINDEX %s ON %s (%s);\n", uniqueStr, c.row["index_name"], c.row["table_name"], c.row["column_names"])
}

// Drop generates SQL to drop the index
func (c IndexSchema) Drop() {
	fmt.Printf("DROP INDEX %s; -- %s ON (%s)\n", c.row["index_name"], c.row["table_name"], c.row["column_names"])
}

// Change handles the case where the table and index name match, but the details do not
func (c IndexSchema) Change(obj interface{}) {
	c2, ok := obj.(*IndexSchema)
	if !ok {
		fmt.Println("Error!!!, change needs an IndexSchema instance", c2)
	}
	// No need to do anything, we either drop or add indices
}

/*
 * Compare the columns in the two databases
 */
func compareIndexes(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT
    t.relname AS table_name,
    i.relname AS index_name,
    ix.indisunique AS unique,
    array_to_string(array_agg(a.attname), ', ') AS column_names
FROM
    pg_index AS ix,
    pg_class AS t,
    pg_class AS i,
    pg_attribute AS a
WHERE
    t.oid = ix.indrelid
    AND i.oid = ix.indexrelid
    AND a.attrelid = t.oid
    AND a.attnum = ANY(ix.indkey)
    AND t.relkind = 'r'
    AND t.relname not like 'pg_%'
GROUP BY t.relname, i.relname, ix.indisunique
ORDER BY t.relname, i.relname, ix.indisunique ASC;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &IndexSchema{channel: rowChan1}
	var schema2 Schema = &IndexSchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
}
