package yaml_test

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
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
	JSON string `json:"json" yaml:"json"`
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
	"————»", "\t",
	"———»", "\t",
	"——»", "\t",
	"—»", "\t",
	"»", "\t",
	"↵", "\n", // is used to show trailing newline characters
	"∎", "", // is used at the end when there is no final newline character
	"←", "\r", // indicates a carriage return character
	"⇔", "\xEF\xBB\xBF", // indicates a byte order mark (BOM) character
)

func readSuite(t require.TestingT) (files []TestFile) {
	a := require.New(t)

	matches, err := fs.Glob(suite, "_testdata/suite/*.yaml")
	a.NoError(err)
	sort.Strings(matches)

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
		if strings.Contains(first.Tags, "1.3") {
			// YAML 1.3 is not supported yet.
			//
			// Skip it early to make test results more clear.
			continue
		}

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

	return files
}

func TestSuite(t *testing.T) {
	files := readSuite(t)

	// tag -> reason
	skipTags := []struct{ tag, reason string }{
		{"libyaml-err", "Skip libyaml error tests"},
		{"empty-key", "Skip empty key tests, libyaml does not support empty keys"},
	}
	// These tests break libyaml.
	//
	// FIXME(tdakkota): Why tf YAML maintainers can't fix their own implementation is a mystery.
	skipFiles := map[string]struct{}{
		// These invalid tests are known to fail.
		"S98Z": {},
		"X4QW": {},
		"SU5Z": {},
		"YJV2": {},
		"CVW2": {},
		"9JBA": {},
		"EB22": {},
		"9HCY": {},
		"G5U8": {},
		"9C9N": {},
		"QB6E": {},
		"RHX7": {},

		// These valid tests are known to fail.
		"W5VH": {},
		"8XYN": {},
		"2SXE": {},
		"7Z25": {},
		"K3WX": {},
		"5MUD": {},
		"5T43": {},
		"QT73": {},
		"MUS6": {},
		"HWV9": {},
		"4ABK": {},
		"VJP3": {},
		"4MUZ": {},
		"58MP": {},
		"96NN": {},
		"9SA2": {},
		"NJ66": {},
		"HM87": {},
		"SM9W": {},
		"6LVF": {},
		"2LFX": {},
		"BEC7": {},
		"A2M4": {},
		"6BCT": {},
		"DBG4": {},
		"R4YG": {},
		"M7A3": {},
		"UT92": {},
		"W4TN": {},
		"Q5MG": {},
		"6CA3": {},
		"Y79Y": {},
		"DK95": {},
		"FP8R": {},
		"DK3J": {},

		// Currently, parser does not accept unknown directives or
		// unsupported version directives.
		"ZYU8": {},
	}

	for _, file := range files {
		file := file
		t.Run(file.TestName, func(t *testing.T) {
			first := file.Tests[0]

			{
				_, fileName := path.Split(file.Name)
				fileName = strings.TrimSuffix(fileName, ".yaml")
				if _, ok := skipFiles[fileName]; ok {
					t.Skipf("Skip %s, known to fail", file.Name)
				}
			}

			for _, skipTag := range skipTags {
				if strings.Contains(first.Tags, skipTag.tag) {
					t.Skipf("Skip %s, %s", file.Name, skipTag.reason)
					return
				}
			}

			for i, test := range file.Tests {
				test := test
				t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
					defer func() {
						r := recover()
						if r != nil || t.Failed() {
							t.Logf("File: %s", file.Name)
							t.Logf("Input: %q", test.YAML)
						}
					}()

					a := require.New(t)

					check := func(s string) error {
						var body yaml.Node
						d := yaml.NewDecoder(strings.NewReader(s))
						for {
							err := d.Decode(&body)
							if err == io.EOF {
								return nil
							}
							if err != nil {
								return err
							}
						}
					}

					err := check(test.YAML)
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
