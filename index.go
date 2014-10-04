package main

import "fmt"
import "strings"
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

// Add generates SQL to add the constraint/index
func (c IndexSchema) Add() {
	fmt.Println("--\n--Add\n--")

	// Assertion
	if c.row["index_def"] == "null" {
		fmt.Printf("-- Unexpected situation in index.go: there is no index_def for %s %s\n", c.row["table_name"], c.row["index_name"])
		return
	}

	// Create the index
	fmt.Printf("%s;\n", c.row["index_def"])

	if c.row["constraint_def"] != "null" {
		// Create the constraint using the index we just created
		if c.row["pk"] == "true" {
			// Add primary key using the index
			fmt.Printf("ALTER TABLE ONLY %s ADD CONSTRAINT %s PRIMARY KEY USING INDEX %s;\n", c.row["table_name"], c.row["index_name"], c.row["index_name"])
		} else if c.row["uq"] == "true" {
			// Add unique constraint using the index
			fmt.Printf("ALTER TABLE ONLY %s ADD CONSTRAINT %s UNIQUE USING INDEX %s;\n", c.row["table_name"], c.row["index_name"], c.row["index_name"])
		}
	}
}

// Drop generates SQL to drop the index and/or the constraint related to it
func (c IndexSchema) Drop() {
	fmt.Println("--\n--Drop\n--")
	if c.row["constraint_def"] != "null" {
		fmt.Println("-- Warning, this may drop foreign keys pointing at this column.  Make sure you re-run the FOREIGN_KEY diff after running this SQL.")
		//fmt.Printf("ALTER TABLE ONLY %s DROP CONSTRAINT IF EXISTS %s CASCADE; -- %s\n", c.row["table_name"], c.row["index_name"], c.row["constraint_def"])
		fmt.Printf("ALTER TABLE ONLY %s DROP CONSTRAINT IF EXISTS %s CASCADE;\n", c.row["table_name"], c.row["index_name"])
	}
	// The second line has no index_def
	//fmt.Printf("DROP INDEX IF EXISTS %s; -- %s \n", c.row["index_name"], c.row["index_def"])
	fmt.Printf("DROP INDEX IF EXISTS %s;\n", c.row["index_name"])
}

// Change handles the case where the table and index name match, but the details do not
func (c IndexSchema) Change(obj interface{}) {
	c2, ok := obj.(*IndexSchema)
	if !ok {
		fmt.Println("-- Error!!!, change needs an IndexSchema instance", c2)
	}
	// Table and constraint name matches... We need to make sure the details match

	// NOTE that there should always be an index_def for both c and c2 (but we're checking below anyway)
	if len(c.row["index_def"]) == 0 {
		fmt.Printf("-- Unexpected situation in index.go: index_def is empty for %v\n", c.row)
		return
	}
	if len(c2.row["index_def"]) == 0 {
		fmt.Printf("-- Unexpected situation in index.go: index_def is empty for %v\n", c2.row)
		return
	}

	if c.row["constraint_def"] != c2.row["constraint_def"] {
		fmt.Println("--\n--CHANGE: constraint defs different\n--")
		// c1.constraint and c2.constraint are just different
		fmt.Printf("-- Different defs:\n--    %s\n--    %s\n", c.row["constraint_def"], c2.row["constraint_def"])
		if c.row["constraint_def"] == "null" {
			// c1.constraint does not exist, c2.constraint does, so
			// Drop constraint
			fmt.Printf("DROP INDEX IF EXISTS %s; -- %s \n", c2.row["index_name"], c2.row["index_def"])
		} else if c2.row["constraint_def"] == "null" {
			// c1.constraint exists, c2.constraint does not, so
			// Add constraint
			if c.row["index_def"] == c2.row["index_def"] {
				// Indexes match, so
				// Add constraint using the index
				if c.row["pk"] == "true" {
					// Add primary key using the index
					fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY USING INDEX %s;\n", c.row["table_name"], c.row["index_name"], c.row["index_name"])
				} else if c.row["uq"] == "true" {
					// Add unique constraint using the index
					fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE USING INDEX %s;\n", c.row["table_name"], c.row["index_name"], c.row["index_name"])
				} else {

				}
			} else {
				// Drop the c2 index, create a copy of the c1 index
				fmt.Printf("DROP INDEX IF EXISTS %s; -- %s \n", c2.row["index_name"], c2.row["index_def"])
			}
			// WIP
			//fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s %s;\n", c.row["table_name"], c.row["index_name"], c.row["constraint_def"])

		} else if c.row["index_def"] != c2.row["index_def"] {
			// The constraints match
		}

	} else if c.row["index_def"] != c2.row["index_def"] {
		if !strings.HasPrefix(c.row["index_def"], c2.row["index_def"]) &&
			!strings.HasPrefix(c2.row["index_def"], c.row["index_def"]) {
			fmt.Println("--\n--Change index defs different\n--")
			// Remember, if we are here, then the two constraint_defs match (both may be empty)
			// The indexes do not match, but the constraints do
			fmt.Printf("--CHANGE: Different index defs:\n--    %s\n--    %s\n", c.row["index_def"], c2.row["index_def"])

			// Drop the index (and maybe the constraint) so we can recreate the index
			c.Drop()

			// Recreate the index (and a constraint if specified)
			c.Add()
		}
	}

}

/*
 * Compare the indexes in the two databases
 */
func compareIndexes(conn1 *sql.DB, conn2 *sql.DB) {
	// This SQL was generated with psql -E -c "\d t_org"
	// The magic is in pg_get_indexdef and pg_get_constraint
	sql := `
SELECT c.relname AS table_name
    , c2.relname AS index_name
    , i.indisprimary AS pk
    , i.indisunique AS uq
    , pg_catalog.pg_get_indexdef(i.indexrelid, 0, true) AS index_def
    , pg_catalog.pg_get_constraintdef(con.oid, true) AS constraint_def
    , con.contype AS typ
FROM pg_catalog.pg_index AS i
JOIN pg_catalog.pg_class AS c ON (c.oid = i.indrelid)
JOIN pg_catalog.pg_class AS c2 ON (c2.oid = i.indexrelid) 
LEFT JOIN pg_catalog.pg_constraint con 
    ON (con.conrelid = i.indrelid AND con.conindid = i.indexrelid AND con.contype IN ('p','u','x'))
WHERE c.relname NOT LIKE 'pg_%'
AND c.relname = 't_org' 
ORDER BY c.relname, c2.relname;
`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &IndexSchema{channel: rowChan1}
	var schema2 Schema = &IndexSchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
}
