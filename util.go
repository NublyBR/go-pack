package pack

import (
	"io"
)

type limitedWriter struct {
	// Original limit
	O uint64

	// Current limit
	N uint64

	// Original writer
	W io.Writer
}

func (l *limitedWriter) Write(b []byte) (n int, err error) {
	if uint64(len(b)) > l.N {
		l.N -= uint64(len(b))
		return 0, &ErrDataTooLarge{max: l.O, size: l.O - l.N}
	}

	n, err = l.W.Write(b)
	l.N -= uint64(n)

	return
}

type limitedReader struct {
	// Original limit
	O uint64

	// Current limit
	N uint64

	// Original reader
	R io.Reader
}

func (l *limitedReader) Read(b []byte) (n int, err error) {
	if l.N == 0 {
		l.N -= uint64(len(b))
		return 0, &ErrDataTooLarge{max: l.O, size: l.O - l.N}
	}

	if uint64(len(b)) > l.N {
		n, err = l.R.Read(b[:l.N])
	} else {
		n, err = l.R.Read(b)
	}

	l.N -= uint64(n)

	return
}
