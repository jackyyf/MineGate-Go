package main

import (
	"testing"
)

func TestVersionString(t *testing.T) {
	if version_short == "" {
		t.Error("Empty short version!")
	} else {
		t.Log("Short version: " + version_short)
	}
	if version_full == "" {
		t.Error("Empty full version!")
	} else {
		t.Log("Short version: " + version_full)
	}

}
