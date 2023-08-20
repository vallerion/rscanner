package rscanner_test

import (
	"bufio"
	"errors"
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

	for {
		e, a := expSc.Scan(), actSc.Scan()

		require.Equal(t, e, a)
		require.Equal(t, expSc.Err(), actSc.Err())
		require.Equal(t, expSc.Bytes(), actSc.Bytes())
		if a == false {
			break
		}
	}
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

func TestScanBufReachMaxTokenSize(t *testing.T) {
	tokenSize, bufSize := 15, 10
	lines := generateLines(tokenSize-1, 3)

	slices.Reverse(lines)
	s := strings.Join(lines, "\n")

	r := strings.NewReader(s)
	sc := rscanner.NewScanner(&slowReaderAt{1, r}, int64(len(s)))
	sc.MaxTokenSize(tokenSize)
	sc.Buffer(make([]byte, bufSize))

	require.True(t, sc.Scan())
	require.NotEmpty(t, sc.Bytes())
	require.False(t, sc.Scan())
	require.ErrorIs(t, sc.Err(), rscanner.ErrTooLong)
}

func TestScanSmallInitBuf(t *testing.T) {
	tokenSize := 10
	n := 101
	lines := generateLines(1, n)

	s := strings.Join(lines, "\n")

	r := strings.NewReader(s)
	sc := rscanner.NewScanner(&slowReaderAt{1, r}, int64(len(s)))
	sc.Buffer(make([]byte, tokenSize))

	for n > 0 {
		require.True(t, sc.Scan())
		require.NotEmpty(t, sc.Bytes())
		require.Nil(t, sc.Err())
		n--
	}

	require.False(t, sc.Scan())
	require.Empty(t, sc.Bytes())
	require.Nil(t, sc.Err())
}

func TestScanZeroInitBuf(t *testing.T) {
	n := 101
	lines := generateLines(1, n)

	s := strings.Join(lines, "\n")

	r := strings.NewReader(s)
	sc := rscanner.NewScanner(&slowReaderAt{1, r}, int64(len(s)))
	sc.Buffer(make([]byte, 0))

	for n > 0 {
		require.True(t, sc.Scan())
		require.NotEmpty(t, sc.Bytes())
		require.Nil(t, sc.Err())
		n--
	}

	require.False(t, sc.Scan())
	require.Empty(t, sc.Bytes())
	require.Nil(t, sc.Err())
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
		p[0] = '\n'
		*r -= negativeEOFReader(c)
		return c, nil
	}
	return -1, io.EOF
}

// Test that the scanner doesn't panic and returns ErrBadReadCount
func TestNegativeEOFReader(t *testing.T) {
	r := negativeEOFReader(12)
	sc := rscanner.NewScanner(&r, 13)
	sc.Buffer(make([]byte, 10))

	require.True(t, sc.Scan())
	require.False(t, sc.Scan())
	require.ErrorIs(t, sc.Err(), rscanner.ErrBadReadCount)
}

// Test that the line splitter handles a final line without a newline.
func testNoNewline(text string, lines []string, t *testing.T) {
	ss := rscanner.NewScanner(&slowReaderAt{7, strings.NewReader(text)}, int64(len(text)))

	for lineNum := 0; ss.Scan(); lineNum++ {
		require.Equal(t, ss.Text(), lines[lineNum])
	}
	require.False(t, ss.Scan())
	require.Nil(t, ss.Err())
}

// Test that the line splitter handles a final line without a newline.
func TestScanLineNoNewline(t *testing.T) {
	const text = "abcdefghijklmn\nopqrstuvwxyz"
	lines := []string{
		"opqrstuvwxyz",
		"abcdefghijklmn",
	}
	testNoNewline(text, lines, t)
}

// Test that the line splitter handles a final line with a carriage return but no newline.
func TestScanLineReturnButNoNewline(t *testing.T) {
	const text = "abcdefghijklmn\nopqrstuvwxyz\r"
	lines := []string{
		"opqrstuvwxyz",
		"abcdefghijklmn",
	}
	testNoNewline(text, lines, t)
}

// Test that the line splitter handles empty line at begin.
func TestScanLineEmptyStartLine(t *testing.T) {
	const text = "\n\nopqrstuvwxyz\nabcdefghijklmn"
	lines := []string{
		"abcdefghijklmn",
		"opqrstuvwxyz",
		"",
	}
	testNoNewline(text, lines, t)
}

// Test that the line splitter handles a final empty line.
func TestScanLineEmptyFinalLine(t *testing.T) {
	const text = "\nopqrstuvwxyz\nabcdefghijklmn\n\n"
	lines := []string{
		"",
		"",
		"abcdefghijklmn",
		"opqrstuvwxyz",
	}
	testNoNewline(text, lines, t)
}

// Test that the line splitter handles a final empty line with a carriage return but no newline.
func TestScanLineEmptyFinalLineWithCR(t *testing.T) {
	const text = "abcdefghijklmn\nopqrstuvwxyz\n\r"
	lines := []string{
		"",
		"opqrstuvwxyz",
		"abcdefghijklmn",
	}
	testNoNewline(text, lines, t)
}

var splitError = errors.New("testError")

// Test the correct error is returned when the split function errors out.
func TestSplitError(t *testing.T) {
	// Create a split function that delivers a little data, then a predictable error.
	numSplits := 0
	const okCount = 7
	errorSplit := func(data []byte) (advance int, token []byte, err error) {
		if numSplits >= okCount {
			return 0, nil, splitError
		}
		numSplits++
		return len(data) - 1, data[len(data)-1:], nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	s := rscanner.NewScanner(&slowReaderAt{1, strings.NewReader(text)}, int64(len(text)))
	s.Split(errorSplit)
	for i := len(text) - 1; i >= len(text)-okCount; i-- {
		require.True(t, s.Scan())
		require.Len(t, s.Bytes(), 1)
		require.Equal(t, text[i], s.Bytes()[0])
	}

	require.False(t, s.Scan())
	require.ErrorIs(t, s.Err(), splitError)
}

// Test the correct error is returned when the split function errors out.
func TestSplitNegativeAdvance(t *testing.T) {
	// Create a split function that delivers a little data, then a predictable error.
	numSplits := 0
	const okCount = 7
	errorSplit := func(data []byte) (advance int, token []byte, err error) {
		if numSplits >= okCount {
			return -1, data[len(data)-1:], nil
		}
		numSplits++
		return len(data) - 1, data[len(data)-1:], nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	s := rscanner.NewScanner(&slowReaderAt{1, strings.NewReader(text)}, int64(len(text)))
	s.Split(errorSplit)

	for i := len(text) - 1; i >= len(text)-okCount; i-- {
		require.True(t, s.Scan())
		require.Len(t, s.Bytes(), 1)
		require.Equal(t, text[i], s.Bytes()[0])
	}

	require.False(t, s.Scan())
	require.ErrorIs(t, s.Err(), rscanner.ErrNegativeAdvance)
}

// Test the correct error is returned when the split function errors out.
func TestSplitAdvanceMoreThanBuffer(t *testing.T) {
	// Create a split function that delivers a little data, then a predictable error.
	numSplits := 0
	const okCount = 7
	const bufSize = 10
	errorSplit := func(data []byte) (advance int, token []byte, err error) {
		if numSplits >= okCount {
			return bufSize + 1, data[len(data)-1:], nil
		}
		numSplits++
		return len(data) - 1, data[len(data)-1:], nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	s := rscanner.NewScanner(&slowReaderAt{1, strings.NewReader(text)}, int64(len(text)))
	s.Split(errorSplit)
	s.Buffer(make([]byte, bufSize))

	for i := len(text) - 1; i >= len(text)-okCount; i-- {
		require.True(t, s.Scan())
		require.Len(t, s.Bytes(), 1)
		require.Equal(t, text[i], s.Bytes()[0])
	}

	require.False(t, s.Scan())
	require.ErrorIs(t, s.Err(), rscanner.ErrAdvanceTooFar)
}

// Test the correct error is returned when the split function errors out.
func TestSplitReturnAlwaysNothing(t *testing.T) {
	maxConsecutiveEmptyReads := 100
	errorSplit := func(data []byte) (advance int, token []byte, err error) {
		return 0, nil, nil
	}
	// Read the data.
	const text = "abcdefghijklmnopqrstuvwxyz"
	s := rscanner.NewScanner(&slowReaderAt{1, strings.NewReader(text)}, int64(len(text)))
	s.Split(errorSplit)
	s.MaxConsecutiveEmptyReads(maxConsecutiveEmptyReads)

	require.True(t, s.Scan())
	require.Equal(t, []byte(text), s.Bytes())
	require.Nil(t, s.Err())
}

type alwaysErrorReaderAt struct{}

func (alwaysErrorReaderAt) ReadAt(p []byte, off int64) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestNonEOFWithEmptyRead(t *testing.T) {
	scanner := rscanner.NewScanner(alwaysErrorReaderAt{}, 10)

	require.False(t, scanner.Scan())
	require.ErrorIs(t, scanner.Err(), io.ErrUnexpectedEOF)
}

// Test that Scan finishes if we have endless empty reads.
type endlessZeros struct{}

func (endlessZeros) ReadAt(p []byte, off int64) (int, error) {
	return 0, nil
}

func TestBadReader(t *testing.T) {
	s := rscanner.NewScanner(endlessZeros{}, 11)
	s.MaxConsecutiveEmptyReads(10)

	require.False(t, s.Scan())
	require.ErrorIs(t, s.Err(), io.ErrNoProgress)
}
