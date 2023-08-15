package rscanner_test

import (
	"bufio"
	"github.com/stretchr/testify/require"
	"github.com/vallerion/rscanner"
	"golang.org/x/exp/slices"
	"io"
	"strings"
	"testing"
)

func TestVsBufioScanner(t *testing.T) {
	l := generateLines(0, 1000)

	act := strings.Join(l, "\n")
	slices.Reverse(l)
	rev := strings.Join(l, "\n")

	expSc := bufio.NewScanner(strings.NewReader(rev))
	actSc := rscanner.NewScanner(strings.NewReader(act), int64(len(act)))

	for i := 0; i < 8; i++ {
		require.Equal(t, expSc.Scan(), actSc.Scan())
		require.Equal(t, expSc.Err(), actSc.Err())
		require.Equal(t, expSc.Bytes(), actSc.Bytes())
	}
	require.Equal(t, expSc.Scan(), actSc.Scan())
	require.Equal(t, expSc.Err(), actSc.Err())
	require.Equal(t, expSc.Bytes(), actSc.Bytes())
}

// slowReader is a reader that returns only a few bytes at a time, to test the incremental
// reads in Scanner.Scan.
type slowReader struct {
	max int
	buf io.ReaderAt
}

func (sr *slowReader) ReadAt(p []byte, off int64) (n int, err error) {
	if len(p) > sr.max {
		p = p[0:sr.max]
	}
	return sr.buf.ReadAt(p, off)
}

func generateLines(minLength, n int) []string {
	lines := make([]string, n)
	for i := minLength; i < minLength+n; i++ {
		lines[i-minLength] = strings.Repeat("U", i)
	}

	return lines
}

func testLines(t *testing.T, tokenSize int, withCR bool) {
	lines := generateLines(0, tokenSize*2)

	var s string
	if withCR {
		s = strings.Join(lines, "\r\n")
	} else {
		s = strings.Join(lines, "\n")
	}

	r := strings.NewReader(s)
	sc := rscanner.NewScanner(r, int64(len(s)))
	sc.MaxTokenSize(tokenSize)

	for i := len(lines) - 1; i >= 0; i-- {
		require.True(t, sc.Scan())
		if len(lines[i]) == 0 {
			require.Equal(t, []byte(nil), sc.Bytes())
		} else {
			require.Equal(t, []byte(lines[i]), sc.Bytes())
		}
		require.Equal(t, lines[i], sc.Text())
	}

	require.False(t, sc.Scan())
	require.Nil(t, sc.Err())
}

func TestScanLines(t *testing.T) {
	for tokenSize := 1; tokenSize < 256; tokenSize++ {
		testLines(t, tokenSize*2, true)
	}
}

//func TestScanTooLong(t *testing.T) {
//	tokenSize := 10
//	lines := generateLines(tokenSize-1, 3)
//
//	slices.Reverse(lines)
//	s := strings.Join(lines, "\n")
//
//	r := strings.NewReader(s)
//	sc := rscanner.NewScanner(&slowReader{1, r}, int64(len(s)))
//	sc.MaxTokenSize(tokenSize)
//	sc.Buffer(make([]byte, tokenSize))
//
//	require.True(t, sc.Scan())
//	require.NotEmpty(t, sc.Bytes())
//	require.False(t, sc.Scan())
//	require.ErrorIs(t, sc.Err(), rscanner.ErrTooLong)
//}
