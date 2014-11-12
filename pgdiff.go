package main

import "fmt"
import "log"
import "flag"
import "os"
import "strings"
import _ "github.com/lib/pq"
import "github.com/joncrlsn/pgutil"

// Schema is a database definition (table, column, constraint, indes, role, etc) that can be
// added, dropped, or changed to match another database.
type Schema interface {
	Compare(schema interface{}) int
	Add()
	Drop()
	Change(schema interface{})
	NextRow() bool
}

var args []string
var dbInfo1 pgutil.DbInfo
var dbInfo2 pgutil.DbInfo
var schemaType string

/*
 * Initialize anything needed later
 */
func init() {
}

/*
 * Do the main logic
 */
func main() {

	dbInfo1, dbInfo2 = parseFlags()
	fmt.Println("-- db1:", dbInfo1)
	fmt.Println("-- db2:", dbInfo2)
	fmt.Println("-- Run the following SQL againt db2:")

	// Remaining args:
	args = flag.Args()
	if len(args) == 0 {
		fmt.Println("The required first argument is SchemaType: SEQUENCE, TABLE, COLUMN, INDEX, FOREIGN_KEY, ROLE, GRANT")
		os.Exit(1)
	}
	schemaType = strings.ToUpper(args[0])

	conn1, err := dbInfo1.Open()
	check("opening database", err)

	conn2, err := dbInfo2.Open()
	check("opening database", err)

	// This section needs to be improved so that you do not need to choose the type
	// of alter statements to generate.  Rather, all should be generated in the
	// proper order.
	if schemaType == "ALL" {
		compareSequences(conn1, conn2)
		compareTables(conn1, conn2)
		compareColumns(conn1, conn2)
		compareIndexes(conn1, conn2) // includes PK and Unique constraints
		compareForeignKeys(conn1, conn2)
		compareRoles(conn1, conn2)
		compareGrants(conn1, conn2)
	} else if schemaType == "SEQUENCE" {
		compareSequences(conn1, conn2)
	} else if schemaType == "TABLE" {
		compareTables(conn1, conn2)
	} else if schemaType == "COLUMN" {
		compareColumns(conn1, conn2)
	} else if schemaType == "INDEX" {
		compareIndexes(conn1, conn2)
	} else if schemaType == "FOREIGN_KEY" {
		compareForeignKeys(conn1, conn2)
	} else if schemaType == "ROLE" {
		compareRoles(conn1, conn2)
	} else if schemaType == "GRANT" {
		compareGrants(conn1, conn2)
	} else {
		fmt.Println("Not yet handled:", schemaType)
	}
}

/*
 * This is a generic diff function that compares tables, columns, indexes, roles, grants, etc.
 * Different behaviors are specified the Schema implementations
 */
func doDiff(db1 Schema, db2 Schema) {

	more1 := db1.NextRow()
	more2 := db2.NextRow()
	for more1 || more2 {
		compareVal := db1.Compare(db2)
		if compareVal == 0 {
			// table and column match, look for non-identifying changes
			db1.Change(db2)
			more1 = db1.NextRow()
			more2 = db2.NextRow()
		} else if compareVal < 0 {
			// db2 is missing a value that db1 has
			if more1 {
				db1.Add()
				more1 = db1.NextRow()
			} else {
				// db1 is at the end
				db2.Drop()
				more2 = db2.NextRow()
			}
		} else if compareVal > 0 {
			// db2 has an extra column that we don't want
			if more2 {
				db2.Drop()
				more2 = db2.NextRow()
			} else {
				// db2 is at the end
				db1.Add()
				more1 = db1.NextRow()
			}
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [database flags] <schemaType> \n", os.Args[0])
	fmt.Fprintln(os.Stderr, `
Compares the schema between two PostgreSQL databases and generates alter statements 
to be *manually* run against the second database.

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

<schemaTpe> : type of schema to check: TABLE, COLUMN, FOREIGN_KEY (soon: CONSTRAINT, ROLE)
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
