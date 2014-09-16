package main

import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"

// ForeignKeySchema holds a channel streaming foreign key data from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// ForeignKeySchema implements the Schema interface defined in pgdiff.go
type ForeignKeySchema struct {
	channel chan map[string]string
	row     map[string]string
	done    bool
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *ForeignKeySchema) NextRow() bool {
	c.row = <-c.channel
	if len(c.row) == 0 {
		c.done = true
	}
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *ForeignKeySchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a ForeignKeySchema instance", c2)
		return +999
	}

	//fmt.Printf("Comparing %s with %s", c.row["table_name"], c2.row["table_name"])
	val := _compareString(c.row["table_name"], c2.row["table_name"])
	if val != 0 {
		return val
	}

	val = _compareString(c.row["constraint_name"], c2.row["constraint_name"])
	return val
}

// Add returns SQL to add the foreign key
func (c ForeignKeySchema) Add() {
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY(%s) REFERENCES %s(%s);\n", c.row["table_name"], c.row["constraint_name"], c.row["column_name"], c.row["foreign_table_name"], c.row["foreign_column_name"])
}

// Drop returns SQL to drop the foreign key
func (c ForeignKeySchema) Drop() {
	fmt.Printf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n", c.row["table_name"], c.row["constraint_name"])
}

// Change handles the case where the table and foreign key name, but the details do not
func (c ForeignKeySchema) Change(obj interface{}) {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, change needs a ForeignKeySchema instance", c2)
	}
	//fmt.Printf("Change Table? %s - %s\n", c.row["table_name"], c2.row["table_name"])
}

/*
 * Compare the columns in the two databases
 */
func compareForeignKeys(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT tc.constraint_name
    , tc.table_name
    , kcu.column_name
    , ccu.table_name AS foreign_table_name
    , ccu.column_name AS foreign_column_name
	, rc.delete_rule AS on_delete
	, rc.update_rule AS on_update
FROM information_schema.table_constraints AS tc
     JOIN information_schema.key_column_usage AS kcu
       ON (tc.constraint_name = kcu.constraint_name)
     JOIN information_schema.constraint_column_usage AS ccu
       ON (ccu.constraint_name = tc.constraint_name)
     JOIN information_schema.referential_constraints rc
       ON (tc.constraint_catalog = rc.constraint_catalog
      AND tc.constraint_schema = rc.constraint_schema
      AND tc.constraint_name = rc.constraint_name)
WHERE tc.constraint_type = 'FOREIGN KEY' 
ORDER BY tc.table_name, tc.constraint_name COLLATE "C" ASC; 


-- Foreign Keys
--SELECT
--    con.relname AS child_table,
--    att2.attname AS child_column, 
--    cl.relname AS parent_table, 
--    att.attname AS parent_column
--FROM
--   (SELECT 
--        unnest(con1.conkey) AS parent, 
--        unnest(con1.confkey) AS child, 
--        cl.relname,
--        con1.confrelid, 
--        con1.conrelid
--    FROM pg_class AS cl
--    JOIN pg_namespace AS ns ON (cl.relnamespace = ns.oid)
--    JOIN pg_constraint AS con1 ON (con1.conrelid = cl.oid)
--    WHERE con1.contype = 'f'
--    --AND cl.relname = 't_org'
--    --AND ns.nspname = 'child_schema'
--   ) con
--JOIN pg_attribute AS att ON (att.attrelid = con.confrelid AND att.attnum = con.child)
--JOIN pg_class AS cl ON (cl.oid = con.confrelid)
--JOIN pg_attribute AS att2 ON (att2.attrelid = con.conrelid AND att2.attnum = con.parent)
--ORDER BY con.relname, att2.attname;

`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &ForeignKeySchema{channel: rowChan1}
	var schema2 Schema = &ForeignKeySchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
}
