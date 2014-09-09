package main

import "fmt"

import "log"
import "flag"
import "os"
import "bytes"
import "strings"
import "database/sql"
import _ "github.com/lib/pq"
import "github.com/joncrlsn/pgutil"


// Schema is a database definition (table, column, constraint, indes, role, etc) that can be
// added, dropped, or changed to match another database.
type Schema interface {
	Compare(schema interface{}) int
	Add()
	Drop()
	Change(schema interface{})
	NextRow(more bool) bool
}

/*
 *
 */
func main() {
	//testDiff()
	//os.Exit(0)

	dbInfo1, dbInfo2 := parseFlags()
	fmt.Println("-- db1:", dbInfo1)
	fmt.Println("-- db2:", dbInfo2)
	fmt.Println("Run the following SQL againt db2")

	// Remaining args:
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("The required first argument is SchemaType: TABLE, COLUMN, FOREIGN_KEY, CONSTRAINT, ROLE")
		os.Exit(1)
	}

	conn1, err := dbInfo1.Open()
	check("opening database", err)

	conn2, err := dbInfo2.Open()
	check("opening database", err)

	// This section will be improved so that you do not need to choose the type
	// of alter statements to generate.  Rather, all should be generated in the
	// proper order.
	schemaType := strings.ToUpper(args[0])
	if schemaType == "TABLE" {
		compareTables(conn1, conn2)
	} else if schemaType == "COLUMN" {
		compareColumns(conn1, conn2)
	} else if schemaType == "FOREIGN_KEY" {
		compareForeignKeys(conn1, conn2)
	} else {
		fmt.Println("Not yet handled:", schemaType)
	}
}

/*
 * Compare the tables in the two databases
 */
func compareTables(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT table_name
    , table_type
    , is_insertable_into
FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER by table_name;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &TableSchema{channel: rowChan1}
	var schema2 Schema = &TableSchema{channel: rowChan2}

	// Compare the tables
	doDiff(schema1, schema2)
}

/*
 * Compare the columns in the two databases
 */
func compareColumns(conn1 *sql.DB, conn2 *sql.DB) {
	sql := `
SELECT table_name
    , column_name
    , data_type
    , is_nullable
    , column_default
    , character_maximum_length
FROM information_schema.columns 
WHERE table_schema = 'public' 
ORDER by table_name, column_name;`

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &ColumnSchema{channel: rowChan1}
	var schema2 Schema = &ColumnSchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
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
FROM
    information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu
      ON tc.constraint_name = kcu.constraint_name
    JOIN information_schema.constraint_column_usage AS ccu
      ON ccu.constraint_name = tc.constraint_name
WHERE constraint_type = 'FOREIGN KEY' 
ORDER BY tc.table_name, tc.constraint_name; `

	rowChan1, _ := pgutil.QueryStrings(conn1, sql)
	rowChan2, _ := pgutil.QueryStrings(conn2, sql)

	// We have to explicitly type this as Schema for some reason
	var schema1 Schema = &ForeignKeySchema{channel: rowChan1}
	var schema2 Schema = &ForeignKeySchema{channel: rowChan2}

	// Compare the columns
	doDiff(schema1, schema2)
}

/*
 * This is a generic diff function that can compare tables, columns, constraints, roles.
 * Different behaviors are specified the Schema implementations
 */
func doDiff(db1 Schema, db2 Schema) {

	more1 := db1.NextRow(true)
	more2 := db2.NextRow(true)
	for more1 || more2 {
		compareVal := db1.Compare(db2)
		if compareVal == 0 {
			// table and column match, look for non-identifying changes
			db1.Change(db2)
			more1 = db1.NextRow(more1)
			more2 = db2.NextRow(more2)
		} else if compareVal < 0 {
			// db2 is missing a value that db1 has
			if more1 {
				db1.Add()
				more1 = db1.NextRow(more1)
			} else {
				// db1 is at the end
				db2.Drop()
				more2 = db2.NextRow(more2)
			}
		} else if compareVal > 0 {
			// db2 has an extra column that we don't want
			if more2 {
				db2.Drop()
				more2 = db2.NextRow(more2)
			} else {
				// db2 is at the end
				db1.Add()
				more1 = db1.NextRow(more1)
			}
		}
	}
}

/*
 * Compares the type of each column and sends back a SQL string if necessary.
 * Currently only varchar length changes are handled
 * The returned SQL should be run on the second database.
 * The table and column are assumed to be the same.
 */
func compareType(row1 map[string]string, row2 map[string]string) string {
	if row1["data_type"] == row2["data_type"] {
		return ""
	}
	var buffer bytes.Buffer
	if row1["data_type"] != row2["data_type"] {
		buffer.WriteString(fmt.Sprintf("-- WARNING: This program does not (yet) handle type changes (%s to %s).\n", row2["data_type"], row1["data_type"]))
	} else if row1["data_type"] == "character varying" {
		if row1["character_maximum_length"] != row2["character_maximum_length"] {
			if row1["character_maximum_length"] < row2["character_maximum_length"] {
				buffer.WriteString("-- WARNING: The next statement will shorten a character varying column.\n")
			}
			buffer.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE character varying(%s);\n", row1["table_name"], row1["column_name"], row1["character_maximum_length"]))
		}
	}

	return buffer.String()
}

/*
 * Compares the default and sends back a SQL string if necessary.
 * The returned SQL should be run on the second database.
 * The table and column are assumed to be the same.
 */
func compareDefault(row1 map[string]string, row2 map[string]string) string {
	if row1["column_default"] == "null" {
		if row2["column_default"] != "null" {
			return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;\n", row1["table_name"], row1["column_name"])
		}
	} else if row1["column_default"] != row2["column_default"] {
		return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DEFAULT %s;\n", row1["table_name"], row1["column_name"], row1["column_default"])
	}
	return ""
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [database flags] <genType> <tableName> <whereClause> \n", os.Args[0])
	fmt.Fprintln(os.Stderr, `
Copies table data as either INSERT or UPDATE statements.

[database flags]: (optional)
  -U1     : postgres user (matches psql flag)
  -h1     : database host -- default is localhost (matches psql flag)
  -p1     : port.  defaults to 5432 (matches psql flag)
  -d1     : database name (matches psql flag)
  -pw1    : password for the postgres user (otherwise you'll be prompted)
  -U2     : postgres user (matches psql flag)
  -h2     : database host -- default is localhost (matches psql flag)
  -p2     : port.  defaults to 5432 (matches psql flag)
  -d2     : database name (matches psql flag)
  -pw2    : password for the postgres user (otherwise you'll be prompted)

<genType>     : type of SQL to generate: insert, update

Database connection information can be specified in two ways:
  * Environment variables
  * Program flags (overrides environment variables.  See above)
  * ~/.pgpass file (for the password)
  * Note that if password is not specified, you will be prompted.

`)

	os.Exit(2)
}

func check(msg string, err error) {
	if err != nil {
		log.Fatal("Error "+msg, err)
	}
}

func _compareString(s1 string, s2 string) int {
	if s1 == s2 {
		return 0
	} else if s1 < s2 {
		return -1
	} else {
		return +1
	}
}
