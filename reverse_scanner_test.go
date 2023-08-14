package rscanner_test

import (
	"bufio"
	"github.com/stretchr/testify/require"
	"github.com/vallerion/rscanner"
	"golang.org/x/exp/slices"
	"strings"
	"testing"
)

func TestVsBufioScanner(t *testing.T) {
	l := generateLines(1000)

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
}

func generateLines(n int) []string {
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = strings.Repeat("U", i)
	}

	return lines
}

func testLines(t *testing.T, tokenSize int, withCR bool) {
	lines := generateLines(tokenSize * 2)

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
}

func TestLines(t *testing.T) {
	for tokenSize := 0; tokenSize < 100; tokenSize++ {
		testLines(t, tokenSize*2, true)
	}
}
