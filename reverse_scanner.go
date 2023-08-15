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
		split:        ScanLines,
		start:        bufSize,
		end:          bufSize,
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
	bufSize      int // needed because reader size could be smaller than default buf size
	start, end   int
	rOffset      int64
	r            io.ReaderAt
	rSize        int64
	split        SplitFunc
	err          error
	needRead     bool
	done         bool
	scanCalled   bool
}

func (bs *Scanner) Scan() bool {
	if bs.done || bs.err != nil {
		bs.token = nil
		bs.start = bs.bufSize
		bs.end = bs.bufSize
		return false
	}
	bs.scanCalled = true

	for {
		if bs.needRead {
			bs.decreaseOffset(int64(bs.start))

			if len(bs.buf) == 0 {
				bs.buf = make([]byte, bs.bufSize)
			}

			off := bs.rOffset
			for left := 0; left < bs.start; {
				n, err := bs.r.ReadAt(bs.buf[left:bs.start], off)
				if n < 0 || n > bs.start {
					bs.setErr(ErrBadReadCount)
					return false
				}
				if err != nil {
					bs.setErr(err)
					return false
				}
				left += n
				off += int64(n)
			}
			bs.start = 0
			bs.needRead = false
		}

		advance, token, err := bs.split(bs.buf[bs.start:bs.end])
		if err != nil {
			// todo ErrFinalToken
			bs.setErr(err)
			return false
		}
		bs.token = token

		if advance < 0 {
			bs.setErr(ErrNegativeAdvance)
			return false
		}

		if advance > bs.end-bs.start {
			bs.setErr(ErrAdvanceTooFar)
			return false
		}

		if advance > 0 || token != nil {
			bs.end = bs.start + advance
			return true
		}

		//if token != nil {
		//	bs.token = token
		//	todo s.empties
		//return true
		//}

		//if bs.err != nil {
		//	// Shut it down.
		//	bs.start = bs.bufSize
		//	bs.end = bs.bufSize
		//	return false
		//}

		if bs.rOffset == 0 {
			bs.token = bytes.Trim(bs.buf[bs.start:bs.end], "\r\n")
			bs.done = true
			return true
		}

		// Here we need more data to be loaded.
		// First we can get some more space in buf by moving bytes in buf.
		// Second we can increase buf size.
		bs.needRead = true

		if bs.end != bs.bufSize {
			d := bs.bufSize - bs.end
			if bs.rOffset < int64(d) {
				d = int(bs.rOffset)
			}
			cleann(bs.buf, bs.start, bs.end)
			copy(bs.buf[bs.start+d:bs.end+d], bs.buf[bs.start:bs.end])
			bs.start += d
			bs.end += d
			cleann(bs.buf, bs.start, bs.end)
		} else {
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
			copy(newBuf[newSize-(bs.end-bs.start):newSize], bs.buf[bs.start:bs.end])
			bs.buf = newBuf
			bs.start = newSize - bs.bufSize
			bs.end = newSize
			bs.bufSize = newSize
		}
	}
}

func (bs *Scanner) decreaseOffset(n int64) {
	bs.rOffset -= n
	if bs.rOffset < 0 {
		bs.rOffset = 0
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

func (bs *Scanner) Buffer(buf []byte) {
	if bs.scanCalled {
		panic("Buffer called after Scan")
	}
	bs.buf = buf[0:cap(buf)]
	bs.bufSize = cap(buf)
	bs.start = bs.bufSize
	bs.end = bs.bufSize
}

func (bs *Scanner) MaxTokenSize(max int) {
	if bs.scanCalled {
		panic("Buffer called after Scan")
	}
	bs.maxTokenSize = max
}

func (bs *Scanner) Split(split SplitFunc) {
	if bs.scanCalled {
		panic("Split called after Scan")
	}
	bs.split = split
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

func cleann(data []byte, start, end int) {
	for i := 0; i < start; i++ {
		data[i] = '_'
	}
	for i := end; i < len(data); i++ {
		data[i] = '_'
	}
}
