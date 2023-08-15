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

// slowReader is a reader that returns only a few bytes at a time, to test the incremental
// reads in Scanner.Scan.
type slowReaderAt struct {
	max int
	buf io.ReaderAt
}

func (sr *slowReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if len(p) > sr.max {
		p = p[0:sr.max]
	}
	return sr.buf.ReadAt(p, off)
}

type slowReader struct {
	max int
	buf io.Reader
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	if len(p) > sr.max {
		p = p[0:sr.max]
	}
	return sr.buf.Read(p)
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

func TestScanVsBufioScanner(t *testing.T) {
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

func TestScanVsBufioScannerSlowReader(t *testing.T) {
	l := generateLines(0, 1000)

	act := strings.Join(l, "\n")
	slices.Reverse(l)
	rev := strings.Join(l, "\n")

	expSc := bufio.NewScanner(&slowReader{buf: strings.NewReader(rev), max: 1})
	actSc := rscanner.NewScanner(&slowReaderAt{buf: strings.NewReader(act), max: 1}, int64(len(act)))

	for i := 0; i < 8; i++ {
		require.Equal(t, expSc.Scan(), actSc.Scan())
		require.Equal(t, expSc.Err(), actSc.Err())
		require.Equal(t, expSc.Bytes(), actSc.Bytes())
	}
	require.Equal(t, expSc.Scan(), actSc.Scan())
	require.Equal(t, expSc.Err(), actSc.Err())
	require.Equal(t, expSc.Bytes(), actSc.Bytes())
}

func TestScanLines(t *testing.T) {
	for tokenSize := 1; tokenSize < 256; tokenSize++ {
		testLines(t, tokenSize*2, true)
	}
}

func TestScanTooLong(t *testing.T) {
	tokenSize := 10
	lines := generateLines(tokenSize-1, 3)

	slices.Reverse(lines)
	s := strings.Join(lines, "\n")

	r := strings.NewReader(s)
	sc := rscanner.NewScanner(&slowReaderAt{1, r}, int64(len(s)))
	sc.MaxTokenSize(tokenSize)
	sc.Buffer(make([]byte, tokenSize))

	require.True(t, sc.Scan())
	require.NotEmpty(t, sc.Bytes())
	require.False(t, sc.Scan())
	require.ErrorIs(t, sc.Err(), rscanner.ErrTooLong)
}

// largeReader returns an invalid count that is larger than the number
// of bytes requested.
type largeReaderAt struct{}

func (largeReaderAt) ReadAt(p []byte, off int64) (int, error) {
	return len(p) + 1, nil
}

// Test that the scanner doesn't panic and returns ErrBadReadCount
func TestLargeReader(t *testing.T) {
	sc := rscanner.NewScanner(largeReaderAt{}, 1000)

	require.False(t, sc.Scan())
	require.ErrorIs(t, sc.Err(), rscanner.ErrBadReadCount)
}

type negativeEOFReader int

func (r *negativeEOFReader) ReadAt(p []byte, off int64) (int, error) {
	if *r > 0 {
		c := int(*r)
		if c > len(p) {
			c = len(p)
		}
		for i := 0; i < c; i++ {
			p[i] = 'a'
		}
		p[c-1] = '\n'
		*r -= negativeEOFReader(c)
		return c, nil
	}
	return -1, io.EOF
}

// Test that the scanner doesn't panic and returns ErrBadReadCount
//func TestNegativeEOFReader(t *testing.T) {
//	r := negativeEOFReader(9)
//	sc := rscanner.NewScanner(&r, 10)
//	require.True(t, sc.Scan())
//	require.True(t, sc.Scan())
//	require.False(t, sc.Scan())
//	require.ErrorIs(t, sc.Err(), rscanner.ErrBadReadCount)
//}
