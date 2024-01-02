package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testTemplates = []string{"base", "fun", "index", "layout", "manageParticipants", "nav", "testAll", "transcluded", "useOther"}

func TestBuildSuccess(t *testing.T) {
	cwd, _ := os.Getwd()
	accessor := NewRealFSAccessor(filepath.Join(cwd, "testdata"))
	file, _ := os.Create(os.DevNull)
	b := NewBuilder(&Flags{}, accessor, file)
	b.build()
	defer os.RemoveAll(filepath.Join(cwd, "testdata", "views"))

	entries, err := os.ReadDir(filepath.Join(cwd, "testdata", "views", "html"))
	if err != nil {
		t.Fatalf("No views directory found")
	}
	if len(entries) != len(testTemplates) {
		t.Fatalf("Not all templates were generated, got %d generated, expected %d", len(entries), len(testTemplates))
	}
	expectedEntries, err := os.ReadDir(filepath.Join(cwd, "testdata", "expected"))
	if err != nil {
		t.Fatalf("Unexpected error: could not find expected directory")
	}
	if len(expectedEntries) != len(testTemplates) {
		t.Fatalf("Unexpected error: did not find the expected templates, got %d generated, expected %d", len(expectedEntries), len(testTemplates))

	}
	var expected map[string]string = make(map[string]string)
	for _, entry := range expectedEntries {
		contents, err := os.ReadFile(filepath.Join(cwd, "testdata", "expected", entry.Name()))
		if err != nil {
			t.Fatalf("Unexpected error: could not load expected template: %s", err.Error())
		}
		expected[entry.Name()] = string(contents)
	}
	for _, entry := range entries {
		contents, err := os.ReadFile(filepath.Join(cwd, "testdata", "views", "html", entry.Name()))
		if err != nil {
			t.Fatalf("Unexpected error: could not load generated template")
		}
		expectedContents := expected[entry.Name()]
		if expectedContents == "" {
			t.Fatalf("No file contents for \"%s\"", entry.Name())
		}
		if bytes.Compare([]byte(expectedContents), contents) != 0 {
			t.Logf("Template \"%s\" did not match", entry.Name())
			t.Log("----- Expected -----")
			t.Log(showWhitespace(expectedContents))
			t.Log("----- Received -----")
			t.Log(showWhitespace(string(contents)))
			// t.FailNow()
		}
	}
}

func showWhitespace(str string) string {
	str = strings.ReplaceAll(str, " ", "•")
	str = strings.ReplaceAll(str, "\t", "›")
	str = strings.ReplaceAll(str, "\n", "⏎\n")
	return str
}
