package yaml_test

import (
	"testing"

	yaml "github.com/go-faster/yamlx"
)

func FuzzUnmarshal(f *testing.F) {
	addFuzzingCorpus(func(data []byte) {
		f.Add(data)
	})

	// TODO(tdakkota): move to addFuzzingCorpus, currently DecodeEncodeDecode fuzzing fails
	//  due to some marshaling issues
	for _, file := range readYAMLSuite(f) {
		for _, test := range file.Tests {
			f.Add([]byte(test.YAML))
			if test.JSON != "" {
				f.Add([]byte(test.JSON))
			}
		}
	}
	for _, tt := range readJSONSuite(f) {
		f.Add(tt.Data)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		t.Run("Node", func(t *testing.T) {
			var n yaml.Node
			_ = yaml.Unmarshal(input, &n)
		})
		t.Run("Interface", func(t *testing.T) {
			var v interface{}
			_ = yaml.Unmarshal(input, &v)
		})
	})
}
