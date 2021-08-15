package main

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"testing"
)

func TestDiffColorize(t *testing.T) {
	old := "Lorem ipsum dolor."
	new := "Lorem dolor sit amet."

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(old, new, false)

	fmt.Println(dmp.DiffPrettyText(diffs))
}
