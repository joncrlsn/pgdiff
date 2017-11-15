package main

import (
	"fmt"
	"testing"
)

func Test_parseAcls(t *testing.T) {
	doParseAcls(t, "user1=rwa/c42", "user1", 3)
	doParseAcls(t, "=arwdDxt/c42", "public", 7)   // first of two lines
	doParseAcls(t, "u3=rwad/postgres", "u3", 4) // second of two lines
	doParseAcls(t, "user2=arwxt/postgres", "user2", 5)
	doParseAcls(t, "", "", 0)
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
