package yaml_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-faster/yaml"
)

func BenchmarkUnmarshal(b *testing.B) {
	// TODO(tdakkota): add more benchmarks.
	input, err := json.Marshal(map[string]any{
		"foo": []string{"bar", "baz"},
		"key": map[string]any{
			"a": "b",
			"c": "d",
		},
	})
	require.NoError(b, err)

	var output map[string]any
	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		output = nil
		if err := yaml.Unmarshal(input, &output); err != nil {
			b.Fatal(err)
		}
	}
}
