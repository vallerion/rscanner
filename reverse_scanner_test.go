package rscanner

import (
	"bufio"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func TestReverseScanner_VsBufioScanner(t *testing.T) {
	const simpleExpected = `FBgn0004635	-0.77
FBgn0051156	-1.77
FBgn0033320	1.15
FBgn0036810	2.08
FBgn0037191	-1.05
FBgn0029994	-1.25
ID	EGF_Baseline`

	expSc := bufio.NewScanner(strings.NewReader(simpleExpected))

	f, err := os.Open("testdata/simple.txt")
	require.NoError(t, err)

	st, err := os.Stat("testdata/simple.txt")
	require.NoError(t, err)

	bs := NewScanner(f, st.Size())

	for i := 0; i < 8; i++ {
		require.Equal(t, expSc.Scan(), bs.Scan())
		require.Equal(t, expSc.Err(), bs.Err())
		require.Equal(t, expSc.Bytes(), bs.Bytes())
	}
}
