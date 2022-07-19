package yaml_test

import (
	"testing"

	yaml "github.com/go-faster/yamlx"
)

func BenchmarkMarshal(b *testing.B) {
	// TODO(tdakkota): add more benchmarks.
	input := map[string][][][]string{
		"a": {{{"a"}}},
		"b": {{{"b", "c"}}},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := yaml.Marshal(input); err != nil {
			b.Fatal(err)
		}
	}
}
