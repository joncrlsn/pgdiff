//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	flag "github.com/ogier/pflag"

	"github.com/joncrlsn/pgutil"
	_ "github.com/lib/pq"
)

// Schema is a database definition (table, column, constraint, indes, role, etc) that can be
// added, dropped, or changed to match another database.
type Schema interface {
	Compare(schema interface{}) int
	Add()
	Drop()
	Change(schema interface{})
	NextRow() bool
}

const (
	version = "0.9.3"
)

var (
	args          []string
	dbInfo1       pgutil.DbInfo
	dbInfo2       pgutil.DbInfo
	excludeChecks map[string]struct{}
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
		fmt.Fprintln(os.Stderr, "The required first argument is one or more comma-separated SchemaTypes: "+strings.Join(schemaTypeOrder, ", "))
		os.Exit(1)
	}

	// Verify schemas
	schemas := dbInfo1.DbSchema + dbInfo2.DbSchema
	if schemas != "**" && strings.Contains(schemas, "*") {
		fmt.Println("If one schema is an asterisk, both must be.")
		os.Exit(1)
	}

	schemaTypes, err := parseSchemaTypes(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	conn1, err := dbInfo1.Open()
	check("opening database 1", err)

	conn2, err := dbInfo2.Open()
	check("opening database 2", err)

	for _, st := range schemaTypes {
		runComparison(st, conn1, conn2)
	}
}

// schemaTypeOrder is the order shown in help / error text.
// validSchemaTypes is the O(1) membership set used by the parser.
// Keep both in sync when adding a new comparator.
var schemaTypeOrder = []string{
	"ALL",
	"SCHEMA",
	"ROLE",
	"ENUM",
	"SEQUENCE",
	"TABLE",
	"TABLE_COLUMN",
	"COLUMN",
	"CHECK",
	"INDEX",
	"VIEW",
	"MATVIEW",
	"FOREIGN_KEY",
	"FUNCTION",
	"TRIGGER",
	"OWNER",
	"GRANT_RELATIONSHIP",
	"GRANT_ATTRIBUTE",
	"EXTENSION",
	"COMMENT",
}

var validSchemaTypes = map[string]struct{}{
	"ALL":                {},
	"SCHEMA":             {},
	"ROLE":               {},
	"ENUM":               {},
	"SEQUENCE":           {},
	"TABLE":              {},
	"TABLE_COLUMN":       {},
	"COLUMN":             {},
	"CHECK":              {},
	"INDEX":              {},
	"VIEW":               {},
	"MATVIEW":            {},
	"FOREIGN_KEY":        {},
	"FUNCTION":           {},
	"TRIGGER":            {},
	"OWNER":              {},
	"GRANT_RELATIONSHIP": {},
	"GRANT_ATTRIBUTE":    {},
	"EXTENSION":          {},
	"COMMENT":            {},
}

// parseSchemaTypes splits a comma-separated arg into upper-cased, trimmed schema type tokens.
// Returns an error if any token is empty or unrecognized so we fail before opening connections.
func parseSchemaTypes(raw string) ([]string, error) {
	parts := strings.Split(strings.ToUpper(raw), ",")
	out := make([]string, 0, len(parts))
	var bad []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := validSchemaTypes[p]; !ok {
			bad = append(bad, p)
			continue
		}
		if p == "ALL" {
			return schemaTypeOrder[1:], nil
		}
		out = append(out, p)
	}
	if len(bad) > 0 {
		return nil, fmt.Errorf("unknown schema type(s): %s. valid options: %s", strings.Join(bad, ", "), strings.Join(schemaTypeOrder, ", "))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no schema types provided")
	}
	return out, nil
}

// runComparison dispatches a single schema type to its comparator.
// ALL runs every comparator in the canonical ordering.
func runComparison(schemaType string, conn1, conn2 *sql.DB) {
	fmt.Printf("-- BEGIN DIFF - %s\n", schemaType)
	switch schemaType {
	case "SCHEMA":
		compareSchematas(conn1, conn2)
	case "ROLE":
		compareRoles(conn1, conn2)
	case "ENUM":
		compareEnums(conn1, conn2)
	case "SEQUENCE":
		compareSequences(conn1, conn2)
	case "TABLE":
		compareTables(conn1, conn2)
	case "COLUMN":
		compareColumns(conn1, conn2)
	case "TABLE_COLUMN":
		compareTableColumns(conn1, conn2)
	case "CHECK":
		compareChecks(conn1, conn2)
	case "INDEX":
		compareIndexes(conn1, conn2)
	case "VIEW":
		compareViews(conn1, conn2)
	case "MATVIEW":
		compareMatViews(conn1, conn2)
	case "FOREIGN_KEY":
		compareForeignKeys(conn1, conn2)
	case "FUNCTION":
		compareFunctions(conn1, conn2)
	case "TRIGGER":
		compareTriggers(conn1, conn2)
	case "OWNER":
		compareOwners(conn1, conn2)
	case "GRANT_RELATIONSHIP":
		compareGrantRelationships(conn1, conn2)
	case "GRANT_ATTRIBUTE":
		compareGrantAttributes(conn1, conn2)
	case "EXTENSION":
		compareExtensions(conn1, conn2)
	case "COMMENT":
		compareComments(conn1, conn2)
	}
	fmt.Printf("-- END DIFF - %s\n", schemaType)
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
	fmt.Fprintf(os.Stderr, "%s - version %s\n", os.Args[0], version)
	fmt.Fprintf(os.Stderr, "usage: %s [<options>] <schemaType> \n", os.Args[0])
	fmt.Fprintln(os.Stderr, `
Compares the schema of a source PostgreSQL database against a destination database
and generates alter statements that can be *manually* run against the destination.

Options:
  -?, --help     : print help information
  -V, --version  : print version information
  --src-url      : source db connection url (postgres://user:pass@host:port/dbname?schema=...&sslmode=disable)
  --src-user     : source db user
  --src-pass     : source db password
  --src-host     : source db host. default is localhost
  --src-port     : source db port. default is 5432
  --src-name     : source db name
  --src-schema   : source schema. default is * (all non-system schemas)
  --src-options  : source db options (eg. sslmode=disable)
  --dst-url      : destination db connection url
  --dst-user     : destination db user
  --dst-pass     : destination db password
  --dst-host     : destination db host. default is localhost
  --dst-port     : destination db port. default is 5432
  --dst-name     : destination db name
  --dst-schema   : destination schema. default is * (all non-system schemas)
  --dst-options  : destination db options
  --exclude-check: comma-separated check constraints to skip (format: schema.table.constraint).
                   useful when src/dst PG versions normalize check expressions differently.

<schemaType> can be one or more (comma-separated) of: ALL, SCHEMA, ROLE, ENUM, SEQUENCE, TABLE, TABLE_COLUMN, COLUMN, CHECK, INDEX, VIEW, MATVIEW, FOREIGN_KEY, FUNCTION, TRIGGER, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE, EXTENSION, COMMENT`)

	os.Exit(2)
}

func check(msg string, err error) {
	if err != nil {
		log.Fatal("Error "+msg, err)
	}
}

// quoteIdent wraps a Postgres identifier (schema, table, column, sequence, ...) in
// double quotes so that case-sensitive names like "userId" survive — unquoted
// identifiers are folded to lowercase by the parser. Embedded double quotes are
// doubled per the SQL spec.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// quoteQualified joins identifiers with `.` and quotes each part. Use it for
// schema.table, schema.table.column, etc.
func quoteQualified(parts ...string) string {
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = quoteIdent(p)
	}
	return strings.Join(out, ".")
}

// defsEqual compares function/view definitions while ignoring per-line leading
// and trailing whitespace. editors and migration tools commonly strip trailing
// whitespace or re-indent on save, which would otherwise cause perpetual diffs
// even when the body is logically identical.
func defsEqual(a, b string) bool {
	return normalizeWhitespace(a) == normalizeWhitespace(b)
}

func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.Join(lines, "\n")
}
