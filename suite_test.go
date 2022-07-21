package yaml_test

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	yaml "github.com/go-faster/yamlx"
)

//go:embed _testdata/suite
var suite embed.FS

type SuiteTest struct {
	Name string `json:"name" yaml:"name"`
	Tags string `json:"tags" yaml:"tags"`
	Fail bool   `json:"fail" yaml:"fail"`
	Skip bool   `json:"skip" yaml:"skip"`
	YAML string `json:"yaml" yaml:"yaml"`
}

type TestFile struct {
	Name     string
	TestName string
	Tests    []SuiteTest
}

// Suite README says (why tf did they do it, nobody wants to read suite tests):
//
// 	The YAML files use a number of non-ascii unicode characters to indicate the presence
//	of certain characters that would be otherwise hard to read.
//
// So, we need to replace them with their ASCII equivalents.
var inputCleaner = strings.NewReplacer(
	"␣", " ", // is used for trailing space characters
	// Hard tabs are reresented by one of: (expanding to 4 spaces)
	"———»", strings.Repeat(" ", 4),
	"——»", strings.Repeat(" ", 4),
	"—»", strings.Repeat(" ", 4),
	"»", strings.Repeat(" ", 4),
	"↵", "\n", // is used to show trailing newline characters
	"∎", "", // is used at the end when there is no final newline character
	"←", "\r", // indicates a carriage return character
	"⇔", "\xEF\xBB\xBF", // indicates a byte order mark (BOM) character
)

func TestSuite(t *testing.T) {
	a := require.New(t)

	matches, err := fs.Glob(suite, "_testdata/suite/*.yaml")
	a.NoError(err)
	sort.Strings(matches)

	var files []TestFile
	for _, match := range matches {
		file, err := suite.ReadFile(match)
		a.NoError(err)

		var test []SuiteTest
		a.NoError(yaml.Unmarshal(file, &test))
		a.NotEmpty(test)

		// Clean up the input.
		for i := range test {
			test[i].YAML = inputCleaner.Replace(test[i].YAML)
		}

		first := test[0]
		for i := 1; i < len(test); i++ {
			test[i].Name = first.Name
			test[i].Tags = first.Tags
			test[i].Skip = first.Skip
		}

		files = append(files, TestFile{
			Name:     match,
			TestName: first.Name,
			Tests:    test,
		})
	}

	for _, file := range files {
		file := file
		t.Run(file.TestName, func(t *testing.T) {
			for i, test := range file.Tests {
				test := test
				t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
					defer func() {
						r := recover()
						if r != nil || t.Failed() || t.Skipped() {
							t.Logf("File: %s", file.Name)
							t.Logf("Input: %q", test.YAML)
						}
					}()
					a := require.New(t)

					if test.Skip {
						t.Skip("Optional test")
					}
					if strings.Contains(test.Tags, "1.3-mod") {
						t.Skip("YAML 1.3 is not supported yet")
					}

					var n yaml.Node
					err := yaml.Unmarshal([]byte(test.YAML), &n)
					if test.Fail {
						a.Error(err, "should fail")
						return
					}
					a.NoError(err)
				})
			}
		})
	}
}
