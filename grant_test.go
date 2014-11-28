package main

import (
	"fmt"
	"testing"
)

func Test_parseAcls(t *testing.T) {
	doParseAcls(t, "c42ro=rwa/c42", "c42ro", 3)
	doParseAcls(t, "=arwdDxt/c42", "public", 7)   // first of two lines
	doParseAcls(t, "c42=rwad/postgres", "c42", 4) // second of two lines
	doParseAcls(t, "user2=arwxt/postgres", "user2", 5)
	doParseAcls(t, "", "", 0)
}

/*
 schema |   type   |               relationship_name               |   relationship_acl   |         column_name          | column_acl
--------+----------+-----------------------------------------------+----------------------+------------------------------+-------------
 public | TABLE    | t_brand                                       | c42ro=r/c42          |                              |
 public | TABLE    | t_brand                                       | office42=arwdDxt/c42 |                              |
 public | TABLE    | t_brand                                       | c42=arwdDxt/c42      |                              |
 public | TABLE    | t_computer                                    | c42=arwdDxt/c42      | active                       | c42ro=r/c42
 public | TABLE    | t_computer                                    | office42=arwdDxt/c42 | active                       | c42ro=r/c42
 public | TABLE    | t_computer                                    | c42=arwdDxt/c42      | address                      | c42ro=r/c42
 public | TABLE    | t_computer                                    | office42=arwdDxt/c42 | address                      | c42ro=r/c42
*/

// Note that these must be sorted for this to work
var relationship1 = []map[string]string{
	{"schema": "public", "type": "TABLE", "relationship_name": "table1", "relationship_acl": "c42=rwa/postgres"},
	{"schema": "public", "type": "TABLE", "relationship_name": "table1", "relationship_acl": "o42=xdwra/postgres"},
	{"schema": "public", "type": "TABLE", "relationship_name": "table2", "relationship_acl": "c42=rwa/postgres"},
}

// Note that these must be sorted for this to work
var relationship2 = []map[string]string{
	{"schema": "public", "relationship_name": "table1", "type": "TABLE", "relationship_acl": "c42=r/postgres"},
	{"schema": "public", "relationship_name": "table2", "type": "TABLE", "relationship_acl": "c42=rwad/postgres"},
}

// Note that these must be sorted for this to work
var attribute1 = []map[string]string{
	{"schema": "public", "type": "TABLE", "relationship_name": "table1", "attribute_name": "column1", "attribute_acl": "c42ro=r/postgres"},
	{"schema": "public", "type": "TABLE", "relationship_name": "table1", "attribute_name": "column1", "attribute_acl": "o42ro=rwa/postgres"},
	{"schema": "public", "type": "TABLE", "relationship_name": "table2", "attribute_name": "column2", "attribute_acl": "c42ro=r/postgres"},
}

// Note that these must be sorted for this to work
var attribute2 = []map[string]string{
	{"schema": "public", "type": "TABLE", "relationship_name": "table1", "attribute_name": "column1", "attribute_acl": "c42ro=r/postgres"},
	{"schema": "public", "type": "TABLE", "relationship_name": "table1", "attribute_name": "column1", "attribute_acl": "o42ro=r/postgres"},
}

func Test_diffGrants(t *testing.T) {
	fmt.Println("-- ==========\n-- Grants - Relationships \n-- ==========")
	var relSchema1 Schema = &GrantRelationshipSchema{rows: relationship1, rowNum: -1}
	var relSchema2 Schema = &GrantRelationshipSchema{rows: relationship2, rowNum: -1}
	doDiff(relSchema1, relSchema2)

	fmt.Println("-- ==========\n-- Grants - Attributes \n-- ==========")
	var attSchema1 Schema = &GrantAttributeSchema{rows: attribute1, rowNum: -1}
	var attSchema2 Schema = &GrantAttributeSchema{rows: attribute2, rowNum: -1}
	doDiff(attSchema1, attSchema2)
}

func doParseAcls(t *testing.T, acl string, expectedRole string, expectedPermCount int) {
	fmt.Println("Testing", acl)
	role, perms := parseAcl(acl)
	if role != expectedRole {
		t.Error("Wrong role parsed: " + role + " instead of " + expectedRole)
	}
	if len(perms) != expectedPermCount {
		t.Error("Incorrect number of permissions parsed: %d instead of %d", len(perms), expectedPermCount)
	}
}
