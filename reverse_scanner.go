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

const defaultBufSize = 4096
const defaultMaxConsecutiveEmptyReads = 100

func NewScanner(r io.ReaderAt, readerSize int64) *Scanner {
	bufSize := defaultBufSize
	if readerSize < int64(bufSize) {
		bufSize = int(readerSize)
	}

	return &Scanner{
		bufSize:                  bufSize,
		maxTokenSize:             bufio.MaxScanTokenSize,
		split:                    ScanLines,
		start:                    bufSize,
		end:                      bufSize,
		rOffset:                  readerSize,
		r:                        r,
		maxConsecutiveEmptyReads: defaultMaxConsecutiveEmptyReads,
	}
}

type Scanner struct {
	maxTokenSize             int         // Maximum size of a token.
	token                    []byte      // Last token returned by split.
	buf                      []byte      // Buffer used as argument to split.
	bufSize                  int         // Size of the buffer.
	start, end               int         // Start and End of data to be scanned in buf.
	rOffset                  int64       // Reader offset.
	r                        io.ReaderAt // The reader provided by the user.
	split                    SplitFunc   // The function to split the tokens, can be provided by user.
	err                      error       // Sticky error.
	done                     bool        // Scan has finished.
	scanCalled               bool        // Scan has been called; buffer is in use.
	maxConsecutiveEmptyReads int         // How many empty r reads allowed.
}

func (bs *Scanner) Scan() bool {
	// First check if scanner is done or there is an error.
	if bs.done || bs.err != nil {
		bs.token = nil
		bs.start = bs.bufSize
		bs.end = bs.bufSize
		return false
	}
	bs.scanCalled = true

	for {
		// Read data if there is unused space in buf before start.
		if bs.start > 0 {
			// Decrease offset to load fill all available buf.
			bs.decreaseOffset(int64(bs.start))

			if len(bs.buf) == 0 {
				bs.buf = make([]byte, bs.bufSize)
			}

			// Here we run for-loop in case if reader is slow.
			// It happens wht it's known the reader has N elements,
			// and we have enough available buffer for N elements,
			// but we read < N.
			// So we run a loop until we fully read it.
			off := bs.rOffset
			for left, emptyReads := 0, 0; left < bs.start; {
				n, err := bs.r.ReadAt(bs.buf[left:bs.start], off)
				// If reader misbehave.
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

				// Track empty reads to avoid endless loop.
				if n == 0 {
					emptyReads++
				}
				if emptyReads > bs.maxConsecutiveEmptyReads {
					bs.setErr(io.ErrNoProgress)
					return false
				}
			}
			bs.start = 0
		}

		advance, token, err := bs.split(bs.buf[bs.start:bs.end])
		if err != nil {
			bs.setErr(err)
			return false
		}
		bs.token = token

		// If split function misbehave.
		if advance < 0 {
			bs.setErr(ErrNegativeAdvance)
			return false
		}

		if advance > bs.end-bs.start {
			bs.setErr(ErrAdvanceTooFar)
			return false
		}

		// If advance>0 and token is nil when token is empty string.
		// If token is not nil and advance=0 when token was found on beginning of the buf.
		if advance > 0 || token != nil {
			bs.end = bs.start + advance
			return true
		}

		if bs.rOffset == 0 {
			if bs.start < bs.end {
				bs.token = bytes.Trim(bs.buf[bs.start:bs.end], "\r\n")
				bs.done = true
				return true
			} else {
				bs.token = nil
				bs.done = true
				return false
			}
		}

		// Here we need more data to be loaded.
		// First we can get some more space in buf by moving bytes in buf
		// from left to right, so more data could be loaded before start.
		// For optimization let's move only then end less than half of buf.
		if bs.end < bs.bufSize/2 {
			d := bs.bufSize - bs.end
			if bs.rOffset < int64(d) {
				d = int(bs.rOffset)
			}
			copy(bs.buf[bs.start+d:bs.end+d], bs.buf[bs.start:bs.end])
			bs.start += d
			bs.end += d
		}

		// Second we can increase buf size.
		if bs.start == 0 {
			if bs.bufSize >= bs.maxTokenSize || bs.bufSize > math.MaxInt/2 {
				bs.setErr(ErrTooLong)
				return false
			}

			newSize := bs.bufSize * 2
			if newSize == 0 {
				newSize = defaultBufSize
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

func (bs *Scanner) MaxConsecutiveEmptyReads(v int) {
	bs.maxConsecutiveEmptyReads = v
}

func ScanLines(data []byte) (advance int, token []byte, err error) {
	if i := bytes.LastIndexByte(data, '\n'); i >= 0 {
		return i, bytes.Trim(data[i:], "\r\n"), nil
	}

	// Request more data.
	return 0, nil, nil
}
