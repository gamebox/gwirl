package pathutils_test

import (
	"testing"

	"github.com/gamebox/gwirl/internal/pathutils"
)

var goPathTests = []struct {
	name     string
	input    string
	expected string
}{
	{"abolute path case", "/home/someuser/myproject/templates/foo.html.gwirl", "/home/someuser/myproject/views/html/foo_gwirl.go"},
	{"relative path case", "./templates/foo.html.gwirl", "views/html/foo_gwirl.go"},
	{"not a gwirl file", "./templates/foo.html", ""},
	{"not in template dir", "./foobar/foo.html.gwirl", ""},
}

func TestPathUtilsGoPath(t *testing.T) {
	config := pathutils.NewPathConfig("templates", "views")
	for _, test := range goPathTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if actual := config.GoPath(test.input); actual != test.expected {
				t.Errorf("Failed: '%s' did not match the expected '%s'", actual, test.expected)
			}
		})
	}
}

var gwirlPathTests = []struct {
    name string
    expected string
    input string
}{
	{"abolute path case", "/home/someuser/myproject/templates/foo.html.gwirl", "/home/someuser/myproject/views/html/foo_gwirl.go"},
	{"relative path case", "templates/foo.html.gwirl", "views/html/foo_gwirl.go"},
	{"not a gwirl file", "", "./views/html/foo.html"},
	{"not in template dir", "", "./foobar/foo_gwirl.go"},
}


func TestPathUtilsGwirlPath(t *testing.T) {
	config := pathutils.NewPathConfig("templates", "views")
	for _, test := range gwirlPathTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if actual := config.GwirlPath(test.input); actual != test.expected {
				t.Errorf("Failed: '%s' did not match the expected '%s'", actual, test.expected)
			}
		})
	}
}
