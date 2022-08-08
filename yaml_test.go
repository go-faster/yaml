package yaml

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_isHashable(t *testing.T) {
	tests := []struct {
		val  interface{}
		want bool
	}{
		// Primitives
		{val: nil, want: true},
		{val: 0, want: true},
		{val: int64(0), want: true},
		{val: "foobar", want: true},
		// Complex types
		{val: struct{ val [1]string }{}, want: true},
		{val: chan int(nil), want: true},
		// Pointers
		{val: new(int), want: true},
		{val: new(int64), want: true},
		{val: new(string), want: true},
		{val: new(struct{ val [1]string }), want: true},
		{val: new(func()), want: true},
		{val: new(chan int), want: true},
		{val: new(map[string]string), want: true},
		{val: new([]string), want: true},

		// Not hashable.
		{val: map[string]string{}, want: false},
		{val: []map[string]string{}, want: false},
		{val: [1]map[string]string{}, want: false},

		{val: []string{}, want: false},
		{val: [1][]string{}, want: false},

		{val: (func())(nil), want: false},
		{val: [1]func(){}, want: false},

		{val: struct{ val map[string]string }{}, want: false},
		{val: struct{ val [0]map[string]string }{}, want: false},
		{val: struct{ val [1]map[string]string }{}, want: false},
		{val: struct{ val []string }{}, want: false},
		{val: struct{ val [0][]string }{}, want: false},
		{val: struct{ val [1][]string }{}, want: false},
		{val: struct{ val func() }{}, want: false},
		{val: struct{ val [0]func() }{}, want: false},
		{val: struct{ val [1]func() }{}, want: false},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("Test%d", i+1), func(t *testing.T) {
			defer func() {
				t.Logf("Type: %T, value: %#v", tt.val, tt.val)
			}()
			a := require.New(t)

			check := a.Panics
			if tt.want {
				check = a.NotPanics
			}
			check(func() {
				_ = map[interface{}]struct{}{
					tt.val: {},
				}
			})

			v := reflect.ValueOf(tt.val)
			a.Equal(tt.want, isHashable(v))
		})
	}
}
