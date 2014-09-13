package main

import "fmt"
import "database/sql"
import "github.com/joncrlsn/pgutil"

// UniqueSchema holds a channel streaming foreign key data from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// UniqueSchema implements the Schema interface defined in pgdiff.go
type UniqueSchema struct {
	channel chan map[string]string
	row     map[string]string
	done    bool
}

// NextRow reads from the channel and tells you if there are (probably) more or not
func (c *UniqueSchema) NextRow() bool {
	c.row = <-c.channel
	if len(c.row) == 0 {
		c.done = true
	}
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *UniqueSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*UniqueSchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a UniqueSchema instance", c2)
		return +999
	}
	val := _compareString(c.row["table_name"], c2.row["table_name"])
	if val != 0 {
		return val
	}
	val = _compareString(c.row["constraint_name"], c2.row["constraint_name"])
	return val
}

// Add returns SQL to add the primary key
func (c UniqueSchema) Add() {
	// ALTER TABLE ONLY t_product ADD CONSTRAINT t_product_pkey PRIMARY KEY (product_id, seq_no);
	// ALTER TABLE ONLY t_product ADD CONSTRAINT t_product_pkey UNIQUE (product_id);
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);\n", c.row["table_name"], c.row["constraint_name"], c.uniqueColumnString())
}

// Drop returns SQL to drop the foreign key
func (c UniqueSchema) Drop() {
	fmt.Printf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n", c.row["table_name"], c.row["constraint_name"])
}

// Change handles the case where the table name matches, but the details do not
func (c UniqueSchema) Change(obj interface{}) {
	c2, ok := obj.(*UniqueSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a UniqueSchema instance", c2)
	}
	pk1 := c.uniqueColumnString()
	pk2 := c.uniqueColumnString()
	if pk1 != pk2 {
		fmt.Printf("-- Warning, primary key is different for table %s  pk1:%s  pk2:%s\n", c.row["table_name"], pk1, pk2)
		fmt.Printf("ALTER TABLE %s DROP CONSTRAINT %s;\n", c.row["table_name"], c2.row["constraint_name"])
		fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);\n", c.row["table_name"], c.row["constraint_name"], c.uniqueColumnString())
	}
}

// uniqueColumnString concatenates the primary key column names into one string.
// It's possible this could be done with SQL, I just haven't figured it out yet
func (c UniqueSchema) uniqueColumnString() string {
	pkey := ""
	for i := 1; i <= 5; i++ {
		colName := fmt.Sprintf("col%d", i)
		col := c.row[colName]
		//fmt.Printf("-- colName: %s  val:'%s'\n", colName, col)
		if len(col) > 0 {
			if len(pkey) > 0 {
				pkey = pkey + ","
			}
			pkey = pkey + col
		}
	}
	return pkey
}

/*
 * Compare the primary keys in the two databases.  This SQL can handle up to 5 columns
 * as part of the primary key
 */
func compareUniqueConstraints(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT tc.table_name
	, kcu.constraint_name
	, MAX(CASE WHEN kcu.ordinal_position = 1 THEN kcu.column_name ELSE '' END) AS col1
	, MAX(CASE WHEN kcu.ordinal_position = 2 THEN kcu.column_name ELSE '' END) AS col2
	, MAX(CASE WHEN kcu.ordinal_position = 3 THEN kcu.column_name ELSE '' END) AS col3
	, MAX(CASE WHEN kcu.ordinal_position = 4 THEN kcu.column_name ELSE '' END) AS col4
	, MAX(CASE WHEN kcu.ordinal_position = 5 THEN kcu.column_name ELSE '' END) AS col5
FROM information_schema.table_constraints AS tc 
LEFT JOIN information_schema.key_column_usage kcu
       ON tc.constraint_catalog = kcu.constraint_catalog
      AND tc.constraint_schema = kcu.constraint_schema
      AND tc.constraint_name = kcu.constraint_name
WHERE tc.constraint_type = 'UNIQUE'
GROUP BY tc.table_name, kcu.constraint_name
ORDER BY tc.table_name, kcu.constraint_name COLLATE "C" ASC;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some unknown reason
	var schema1 Schema = &UniqueSchema{channel: rowChan1}
	var schema2 Schema = &UniqueSchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
}
