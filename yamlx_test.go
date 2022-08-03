package yaml_test

import (
	"io"
	"testing"
	"testing/iotest"
)

func testReader(t *testing.T, test func(wrap func(io.Reader) io.Reader) func(t *testing.T)) {
	t.Run("Plain", test(func(r io.Reader) io.Reader { return r }))
	t.Run("OneByteReader", test(iotest.OneByteReader))
	t.Run("DataErrReader", test(iotest.DataErrReader))
	t.Run("HalfReader", test(iotest.HalfReader))
}
