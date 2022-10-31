package yaml_test

import (
	"fmt"
	"io"
	"testing"

	yaml "github.com/go-faster/yamlx"
)

func generateInput() any {
	m := map[string]any{}
	for i := 0; i < 100; i++ {
		m[fmt.Sprintf("foo_%d", i)] = []any{
			map[string]any{
				"bar": "baz",
			},
			i,
		}
	}
	return m
}

func BenchmarkMarshal(b *testing.B) {
	// TODO(tdakkota): add more benchmarks.
	input := generateInput()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := yaml.Marshal(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncoder_Encode(b *testing.B) {
	e := yaml.NewEncoder(io.Discard)
	input := generateInput()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := e.Encode(input); err != nil {
			b.Fatal(err)
		}
	}
}
