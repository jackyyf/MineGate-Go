package main

import (
	"testing"
)

func TestVersionString(t *testing.T) {
	t.Log("Short version: " + version_short)
	t.Log("Full version: " + version_full)
}
