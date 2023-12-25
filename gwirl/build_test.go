package main

import (
	"os"
	"path/filepath"
	"testing"
)

var testTemplates = []string{"base", "layout", "manageParticipants", "nav", "testAll", "transcluded", "useOther"}

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
		actualContents := string(contents)
		if actualContents != expectedContents {
			t.Fatalf("Template \"%s\" did not match", entry.Name())
		}
	}
}
