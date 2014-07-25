package main

import (
	"strings"
	"testing"
)

func error_should_contain(t *testing.T, err error, expected string) {
	if err == nil {
		t.Errorf("Expected error containing:\n\t'%v',\nbut got nil error.", expected)
	} else if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error containing:\n\t'%v',\nbut got\n\t'%v'.", expected, err.Error())
	}
}

type test_base struct{}

func (test_base) ChildResources() []Resource { return nil }
