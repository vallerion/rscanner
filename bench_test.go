package rscanner_test

import (
	"bufio"
	"github.com/icza/backscanner"
	"github.com/vallerion/rscanner"
	"strings"
	"testing"
)

func init() {
	ll := generateLines(1, 10000)
	text = strings.Join(ll, "\n")
}

var text, t string

func BenchmarkScan(b *testing.B) {
	s := rscanner.NewScanner(strings.NewReader(text), int64(len(text)))
	n := 0
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for s.Scan() {
			t = s.Text()
			n++
		}
	}

	if n == 0 {
		b.Fatalf("scanner didn't run")
	}
}

func BenchmarkScanSlowReader(b *testing.B) {
	s := rscanner.NewScanner(&slowReaderAt{max: 1, buf: strings.NewReader(text)}, int64(len(text)))
	n := 0
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for s.Scan() {
			t = s.Text()
			n++
		}
	}

	if n == 0 {
		b.Fatalf("scanner didn't run")
	}
}

func BenchmarkScanBufio(b *testing.B) {
	s := bufio.NewScanner(strings.NewReader(text))
	n := 0
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for s.Scan() {
			t = s.Text()
			n++
		}
	}

	if n == 0 {
		b.Fatalf("scanner didn't run")
	}
}

func BenchmarkScanBufioSlowReader(b *testing.B) {
	s := bufio.NewScanner(&slowReader{max: 1, buf: strings.NewReader(text)})
	n := 0
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for s.Scan() {
			t = s.Text()
			n++
		}
	}

	if n == 0 {
		b.Fatalf("scanner didn't run")
	}
}

func BenchmarkScanBackscanner(b *testing.B) {
	scanner := backscanner.New(strings.NewReader(text), len(text))
	n := 0
	var err error
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for {
			t, _, err = scanner.Line()
			n++
			if err != nil {
				break
			}
		}
	}

	if n == 0 {
		b.Fatalf("scanner didn't run")
	}
}

func BenchmarkScanBackscannerSlowReader(b *testing.B) {
	scanner := backscanner.New(&slowReaderAt{max: 1, buf: strings.NewReader(text)}, len(text))
	n := 0
	var err error
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for {
			t, _, err = scanner.Line()
			n++
			if err != nil {
				break
			}
		}
	}

	if n == 0 {
		b.Fatalf("scanner didn't run")
	}
}
