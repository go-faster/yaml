package yaml

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_keyList(t *testing.T) {
	val := 3
	var (
		sorted = []any{
			false,
			true,
			1,
			uint(1),
			1.0,
			1.1,
			1.2,
			2,
			uint(2),
			2.0,
			2.1,
			&val,
			"",
			".1",
			".2",
			".a",
			"1",
			"2",
			"a!10",
			"a/0001",
			"a/002",
			"a/3",
			"a/10",
			"a/11",
			"a/0012",
			"a/100",
			"a~10",
			"ab/1",
			"b/1",
			"b/01",
			"b/2",
			"b/02",
			"b/3",
			"b/03",
			"b1",
			"b01",
			"b3",
			"c2.10",
			"c10.2",
			"d1",
			"d7",
			"d7abc",
			"d12",
			"d12a",
			"e2b",
			"e4b",
			"e21a",
		}
		list keyList
	)
	for _, v := range sorted {
		list = append(list, reflect.ValueOf(v))
	}

	// Shuffle and sort multiple times to ensure that the sort is stable.
	for i := 0; i < 10; i++ {
		// Randomize the list.
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(list.Len(), func(i, j int) {
			list.Swap(i, j)
		})
		// Sort the list.
		sort.Sort(list)

		got := make([]any, len(list))
		for i, v := range list {
			got[i] = v.Interface()
		}
		require.Equal(t, sorted, got)
	}
}

func Test_numLess(t *testing.T) {
	tests := []struct {
		a, b any
		want bool
	}{
		{0, 0, false},
		{false, false, false},
		{true, true, false},

		{false, true, true},
		{0, 1, true},
		{0, int8(1), true},
		{int8(0), 1, true},
		{uint(0), uint8(1), true},
		{uint8(0), uint(1), true},
		{float32(0), float64(1), true},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			got := numLess(reflect.ValueOf(tt.a), reflect.ValueOf(tt.b))
			require.Equal(t, tt.want, got)
		})
	}
}
