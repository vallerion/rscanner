package rscanner

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"math"
)

type SplitFunc func(data []byte) (advance int, token []byte, err error)

var (
	ErrTooLong         = errors.New("rscanner.Scanner: token too long")
	ErrNegativeAdvance = errors.New("rscanner.Scanner: SplitFunc returns negative advance count")
	ErrAdvanceTooFar   = errors.New("rscanner.Scanner: SplitFunc returns advance count beyond input")
	ErrBadReadCount    = errors.New("rscanner.Scanner: Read returned impossible count")
)

const StartBufSize = 4096

func NewScanner(r io.ReaderAt, readerSize int64) *Scanner {
	bufSize := StartBufSize
	if readerSize < int64(bufSize) {
		bufSize = int(readerSize)
	}

	return &Scanner{
		bufSize:      bufSize,
		maxTokenSize: bufio.MaxScanTokenSize,
		splitFunc:    ScanLines,
		start:        bufSize - 1,
		end:          bufSize - 1,
		rSize:        readerSize,
		rOffset:      readerSize,
		r:            r,
		needRead:     true,
	}
}

type Scanner struct {
	maxTokenSize int
	token        []byte
	buf          []byte
	bufSize      int
	start, end   int

	rOffset int64
	r       io.ReaderAt
	rSize   int64

	splitFunc SplitFunc

	err      error
	needRead bool
	done     bool
}

func (bs *Scanner) Scan() bool {
	if bs.done {
		bs.token = nil
		return false
	}

	for {
		bs.rOffset -= int64(bs.start) + 1
		if bs.rOffset < 0 {
			bs.rOffset = 0
		}

		if bs.needRead {
			if len(bs.buf) == 0 {
				bs.buf = make([]byte, bs.bufSize)
			}

			n, err := bs.r.ReadAt(bs.buf[0:bs.start+1], bs.rOffset)
			if err != nil {
				bs.setErr(err)
				return false
			}
			bs.start = 0
			if n < bs.end-bs.start {
				bs.end = bs.start + n
			}
			if n > bs.end-bs.start+1 {
				bs.setErr(ErrBadReadCount)
				return false
			}
			bs.needRead = false
		}

		advance, token, err := ScanLines(bs.buf[bs.start : bs.end+1])
		if err != nil {
			// todo ErrFinalToken
			bs.setErr(err)
			return false
		}

		if advance < 0 {
			bs.setErr(ErrNegativeAdvance)
			return false
		}

		if advance > bs.end-bs.start {
			bs.setErr(ErrAdvanceTooFar)
			return false
		}

		if advance > 0 {
			bs.end = bs.start + advance - 1
		}

		if token != nil {
			bs.token = token
			// todo s.empties
			return true
		}

		if bs.err != nil {
			// Shut it down.
			bs.start = bs.bufSize - 1
			bs.end = bs.bufSize - 1
			return false
		}

		if bs.rOffset == 0 {
			bs.token = bs.buf[bs.start : bs.end+1]
			bs.done = true
			return true
		}

		// Here we need more data to be loaded.
		// First we can get some more space in buf by moving bytes in buf.
		// Second we can increase buf size.
		bs.needRead = true

		if bs.end != bs.bufSize-1 {
			d := bs.bufSize - bs.end - 1
			copy(bs.buf[bs.start+d:bs.end+d+1], bs.buf[bs.start:bs.end+1])
			bs.start += d
			bs.end += d
		}

		if bs.end == bs.bufSize-1 {
			if bs.bufSize >= bs.maxTokenSize || bs.bufSize > math.MaxInt/2 {
				bs.setErr(ErrTooLong)
				return false
			}

			newSize := bs.bufSize * 2
			if newSize == 0 {
				newSize = StartBufSize
			}
			if newSize > bs.maxTokenSize {
				newSize = bs.maxTokenSize
			}
			newBuf := make([]byte, newSize)
			copy(newBuf[newSize-(bs.end-bs.start)-1:newSize], bs.buf[bs.start:bs.end+1])
			bs.buf = newBuf
			bs.start = newSize - bs.bufSize
			bs.end = newSize - 1
			bs.bufSize = newSize
		}
	}
}

func (bs *Scanner) Err() error {
	return bs.err
}

func (bs *Scanner) Bytes() []byte {
	return bs.token
}

func (bs *Scanner) Text() string {
	return string(bs.token)
}

func (bs *Scanner) setErr(err error) {
	if bs.err == nil {
		bs.err = err
	}
}

func ScanLines(data []byte) (advance int, token []byte, err error) {
	if i := bytes.LastIndexByte(data, '\n'); i >= 0 {
		return i, bytes.Trim(data[i:], "\r\n"), nil
	}

	// Request more data.
	return 0, nil, nil
}
