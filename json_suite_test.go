package yaml_test

import (
	"embed"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-faster/jx"

	"github.com/go-faster/yaml"
)

//go:embed testdata/json_suite
var jsonSuite embed.FS

type JSONSuiteTest struct {
	Name   string
	Action JSONSuiteAction
	Data   []byte
}

type JSONSuiteAction string

const (
	Accept    JSONSuiteAction = "y_"
	Reject    JSONSuiteAction = "n_"
	Undefined JSONSuiteAction = "i_"
)

func readJSONSuite(t require.TestingT) (r []JSONSuiteTest) {
	// https://github.com/nst/JSONTestSuite
	// By Nicolas Seriot (https://github.com/nst)
	a := require.New(t)

	dir := path.Join("testdata", "json_suite")
	files, err := jsonSuite.ReadDir(dir)
	a.NoError(err)
	a.NotEmpty(files)

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(f.Name(), ".json")
		action := JSONSuiteAction(f.Name()[:2])
		a.Contains([]JSONSuiteAction{Accept, Reject, Undefined}, action)

		file := path.Join(dir, f.Name())
		data, err := jsonSuite.ReadFile(file)
		a.NoError(err)

		r = append(r, JSONSuiteTest{
			Name:   name,
			Action: action,
			Data:   data,
		})
	}

	return r
}

func TestJSONSuite(t *testing.T) {
	// Time to break the big lie about YAML and JSON compatibility.
	skipControlCharacters := map[string]struct{}{
		"y_string_nonCharacterInUTF-8_U+FFFF": {},
		"y_string_unescaped_char_delete":      {},
		"y_string_with_del_character":         {},
	}

	for _, tt := range readJSONSuite(t) {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			if _, ok := skipControlCharacters[tt.Name]; ok {
				t.Skip("YAML does not allow control characters in strings")
				return
			}
			a := assert.New(t)
			data := tt.Data

			var n yaml.Node
			err := yaml.Unmarshal(data, &n)
			switch action := tt.Action; action {
			case Accept:
				a.NoError(err, "%#v", string(data))
				e := jx.GetEncoder()
				a.NoError(n.EncodeJSON(e))
				a.JSONEq(string(data), e.String())
			case Undefined, Reject: // Actually, some invalid JSON is accepted by the YAML parser.
				if err == nil {
					t.Log("Accept")
				} else {
					t.Logf("Reject: %v", err)
				}
			default:
				t.Fatalf("Unknown prefix %q", action)
			}
		})
	}
}
