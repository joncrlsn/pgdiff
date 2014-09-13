package main

import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"

// TableSchema holds a channel streaming table information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// TableSchema implements the Schema interface defined in pgdiff.go
type TableSchema struct {
	channel chan map[string]string
	row     map[string]string
	done    bool
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *TableSchema) NextRow() bool {
	c.row = <-c.channel
	if len(c.row) == 0 {
		c.done = true
	}
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *TableSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a TableSchema instance", c2)
		return +999
	}

	//fmt.Printf("Comparing %s with %s", c.row["table_name"], c2.row["table_name"])
	val := _compareString(c.row["table_name"], c2.row["table_name"])
	return val
}

// Add returns SQL to add the table
func (c TableSchema) Add() {
	fmt.Printf("CREATE TABLE %s();\n", c.row["table_name"])
}

// Drop returns SQL to drop the table
func (c TableSchema) Drop() {
	fmt.Printf("DROP TABLE IF EXISTS %s;\n", c.row["table_name"])
}

// Change handles the case where the table and column match, but the details do not
func (c TableSchema) Change(obj interface{}) {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a TableSchema instance", c2)
	}
	//fmt.Printf("Change Table? %s - %s\n", c.row["table_name"], c2.row["table_name"])
}

// compareTables outputs SQL to make the table names match between DBs
func compareTables(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT table_name
    , table_type
    , is_insertable_into
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_type = 'BASE TABLE'
ORDER BY table_name COLLATE "C" ASC;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &TableSchema{channel: rowChan1}
	var schema2 Schema = &TableSchema{channel: rowChan2}

	// Compare the tables
	doDiff(schema1, schema2)
}
