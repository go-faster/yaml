package yaml_test

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/jx"

	"github.com/go-faster/yaml"
)

//go:embed testdata/yaml_suite
var yamlSuite embed.FS

type YAMLSuiteTest struct {
	Name string `json:"name" yaml:"name"`
	Tags string `json:"tags" yaml:"tags"`
	Fail bool   `json:"fail" yaml:"fail"`
	Skip bool   `json:"skip" yaml:"skip"`
	YAML string `json:"yaml" yaml:"yaml"`
	JSON string `json:"json" yaml:"json"`
}

type YAMLSuiteFile struct {
	Name     string
	TestName string
	Tests    []YAMLSuiteTest
}

// Suite README says (why tf did they do it, nobody wants to read suite tests):
//
//	The YAML files use a number of non-ascii unicode characters to indicate the presence
//	of certain characters that would be otherwise hard to read.
//
// So, we need to replace them with their ASCII equivalents.
//
// https://github.com/yaml/yaml-test-suite/blob/main/bin/YAMLTestSuite.pm#L103-L115.
var inputCleaner = strings.NewReplacer(
	"␣", " ", // is used for trailing space characters
	// Hard tabs are reresented by one of: (expanding to 4 spaces)
	"————»", "\t",
	"———»", "\t",
	"——»", "\t",
	"—»", "\t",
	"»", "\t",
	"↵", "", // is used to show trailing newline characters
	"∎", "", // is used at the end when there is no final newline character
	"←", "\r", // indicates a carriage return character
	"⇔", "\xFE\xFF", // indicates a byte order mark (BOM) character
)

func readYAMLSuite(t require.TestingT) (r []YAMLSuiteFile) {
	a := require.New(t)

	dir := path.Join("testdata", "yaml_suite")
	files, err := yamlSuite.ReadDir(dir)
	a.NoError(err)
	a.NotEmpty(files)

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		file := path.Join(dir, f.Name())
		data, err := yamlSuite.ReadFile(file)
		a.NoError(err)

		var test []YAMLSuiteTest
		a.NoError(yaml.Unmarshal(data, &test))
		a.NotEmpty(test)

		// Clean up the input.
		for i := range test {
			test[i].YAML = inputCleaner.Replace(test[i].YAML)
		}

		first := test[0]
		if strings.Contains(first.Tags, "1.3-mod") {
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

		r = append(r, YAMLSuiteFile{
			Name:     file,
			TestName: first.Name,
			Tests:    test,
		})
	}

	return r
}

func TestYAMLSuite(t *testing.T) {
	files := readYAMLSuite(t)

	// tag -> reason
	skipTags := []struct{ tag, reason string }{
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
	skipJSON := map[string]struct{}{
		// Scanner assumes that '?' is not part of a key.
		"652Z": {},
		// Parser/Decoder does not pass tag information, so we encode value as string, but integer is expected.
		"S4JQ": {},
	}

	for _, file := range files {
		file := file
		t.Run(file.TestName, func(t *testing.T) {
			first := file.Tests[0]
			_, fileName := path.Split(file.Name)
			fileName = strings.TrimSuffix(fileName, ".yaml")

			{
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

					check := func(s string) (r []yaml.Node, _ error) {
						d := yaml.NewDecoder(strings.NewReader(s))
						for {
							var doc yaml.Node
							err := d.Decode(&doc)
							if err == io.EOF {
								return r, nil
							}
							if err != nil {
								return nil, err
							}
							r = append(r, doc)
						}
					}

					docs, err := check(test.YAML)
					if test.Fail {
						a.Error(err, "should fail")
						return
					}
					a.NoError(err)

					if _, ok := skipJSON[fileName]; !ok && test.JSON != "" {
						var expected []json.RawMessage
						d := json.NewDecoder(strings.NewReader(test.JSON))
						for {
							var doc json.RawMessage
							err := d.Decode(&doc)
							if err == io.EOF {
								break
							}
							a.NoError(err)
							expected = append(expected, doc)
						}
						a.Equal(len(expected), len(docs))

						for i, doc := range docs {
							jsonDoc := expected[i]

							var e jx.Encoder
							a.NoError(doc.EncodeJSON(&e))
							a.JSONEq(string(jsonDoc), e.String())
						}
					}
				})
			}
		})
	}
}
