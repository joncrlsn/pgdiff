//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import "fmt"
import "log"
import flag "github.com/ogier/pflag"
import "os"
import "strings"
import _ "github.com/lib/pq"
import "github.com/joncrlsn/pgutil"

// Schema is a database definition (table, column, constraint, indes, role, etc) that can be
// added, dropped, or changed to match another database.
type Schema interface {
	Compare(schema interface{}) int
	Add(schema interface{})
	Drop(schema interface{})
	Change(schema interface{})
	NextRow() bool
}

const (
	version = "0.9.3"
)

var (
	args       []string
	dbInfo1    pgutil.DbInfo
	dbInfo2    pgutil.DbInfo
	schemaType string
)

/*
 * Initialize anything needed later
 */
func init() {
}

/*
 * Do the main logic
 */
func main() {

	var helpPtr = flag.BoolP("help", "?", false, "print help information")
	var versionPtr = flag.BoolP("version", "V", false, "print version information")

	dbInfo1, dbInfo2 = parseFlags()

	// Remaining args:
	args = flag.Args()

	if *helpPtr {
		usage()
	}

	if *versionPtr {
		fmt.Fprintf(os.Stderr, "%s - version %s\n", os.Args[0], version)
		fmt.Fprintln(os.Stderr, "Copyright (c) 2017 Jon Carlson.  All rights reserved.")
		fmt.Fprintln(os.Stderr, "Use of this source code is governed by the MIT license")
		fmt.Fprintln(os.Stderr, "that can be found here: http://opensource.org/licenses/MIT")
		os.Exit(1)
	}

	if len(args) == 0 {
		fmt.Println("The required first argument is SchemaType: SCHEMA, ROLE, SEQUENCE, TABLE, VIEW, COLUMN, INDEX, FOREIGN_KEY, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE")
		os.Exit(1)
	}

	// Verify schemas
	schemas := dbInfo1.DbSchema + dbInfo2.DbSchema
	if schemas != "**" && strings.Contains(schemas, "*") {
		fmt.Println("If one schema is an asterisk, both must be.")
		os.Exit(1)
	}

	schemaType = strings.ToUpper(args[0])
	fmt.Println("-- schemaType:", schemaType)

	fmt.Println("-- db1:", dbInfo1)
	fmt.Println("-- db2:", dbInfo2)
	fmt.Println("-- Run the following SQL against db2:")

	conn1, err := dbInfo1.Open()
	check("opening database 1", err)

	conn2, err := dbInfo2.Open()
	check("opening database 2", err)

	// This section needs to be improved so that you do not need to choose the type
	// of alter statements to generate.  Rather, all should be generated in the
	// proper order.
	if schemaType == "ALL" {
		if dbInfo1.DbSchema == "*" {
			compareSchematas(conn1, conn2)
		}
		compareRoles(conn1, conn2)
		compareFunctions(conn1, conn2)
		compareSchematas(conn1, conn2)
		compareSequences(conn1, conn2)
		compareTables(conn1, conn2)
		compareColumns(conn1, conn2)
		compareIndexes(conn1, conn2) // includes PK and Unique constraints
		compareViews(conn1, conn2)
		compareOwners(conn1, conn2)
		compareForeignKeys(conn1, conn2)
		compareGrantRelationships(conn1, conn2)
		compareGrantAttributes(conn1, conn2)
		compareTriggers(conn1, conn2)
	} else if schemaType == "SCHEMA" {
		compareSchematas(conn1, conn2)
	} else if schemaType == "ROLE" {
		compareRoles(conn1, conn2)
	} else if schemaType == "SEQUENCE" {
		compareSequences(conn1, conn2)
	} else if schemaType == "TABLE" {
		compareTables(conn1, conn2)
	} else if schemaType == "COLUMN" {
		compareColumns(conn1, conn2)
	} else if schemaType == "FUNCTION" {
		compareFunctions(conn1, conn2)
	} else if schemaType == "VIEW" {
		compareViews(conn1, conn2)
	} else if schemaType == "INDEX" {
		compareIndexes(conn1, conn2)
	} else if schemaType == "FOREIGN_KEY" {
		compareForeignKeys(conn1, conn2)
	} else if schemaType == "OWNER" {
		compareOwners(conn1, conn2)
	} else if schemaType == "GRANT_RELATIONSHIP" {
		compareGrantRelationships(conn1, conn2)
	} else if schemaType == "GRANT_ATTRIBUTE" {
		compareGrantAttributes(conn1, conn2)
	} else if schemaType == "TRIGGER" {
		compareTriggers(conn1, conn2)
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
				db1.Add(db2)
				more1 = db1.NextRow()
			} else {
				// db1 is at the end
				db2.Drop(db2)
				more2 = db2.NextRow()
			}
		} else if compareVal > 0 {
			// db2 has an extra column that we don't want
			if more2 {
				db2.Drop(db2)
				more2 = db2.NextRow()
			} else {
				// db2 is at the end
				db1.Add(db2)
				more1 = db1.NextRow()
			}
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s - version %s\n", os.Args[0], version)
	fmt.Fprintf(os.Stderr, "usage: %s [<options>] <schemaType> \n", os.Args[0])
	fmt.Fprintln(os.Stderr, `
Compares the schema between two PostgreSQL databases and generates alter statements 
that can be *manually* run against the second database.

Options:
  -?, --help    : print help information
  -V, --version : print version information
  -v, --verbose : print extra run information
  -U, --user1   : first postgres user 
  -u, --user2   : second postgres user 
  -H, --host1   : first database host.  default is localhost 
  -h, --host2   : second database host. default is localhost 
  -P, --port1   : first port.  default is 5432 
  -p, --port2   : second port. default is 5432 
  -D, --dbname1 : first database name 
  -d, --dbname2 : second database name 
  -S, --schema1 : first schema.  default is public
  -s, --schema2 : second schema. default is public

<schemaTpe> can be: SCHEMA ROLE, SEQUENCE, TABLE, VIEW, COLUMN, INDEX, FOREIGN_KEY, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE
`)

	os.Exit(2)
}

func check(msg string, err error) {
	if err != nil {
		log.Fatal("Error "+msg, err)
	}
}
