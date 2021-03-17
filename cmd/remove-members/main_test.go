package main

import (
	"reflect"
	"testing"
)

func TestReadMemberList(t *testing.T) {
	memberList, err := readMemberList("test/members.txt")
	if err != nil {
		t.Error(err)
	}

	expectedMemberList := []string{"johndoe", "janedoe"}

	if !reflect.DeepEqual(memberList, expectedMemberList) {
		t.Errorf("Values mismatch. Expected: %v Actual: %v", expectedMemberList, memberList)
	}
}

func TestRemoveMembers(t *testing.T) {

	memberList := []string{"johndoe", "janedoe"}

	if err := removeMembers(memberList, "test"); err != nil {
		t.Error(err)
	}
}
