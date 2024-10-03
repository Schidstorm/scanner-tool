package server

import "io"

type bytesReadSeeker struct {
	data []byte
	pos  int
}

func newBytesReadSeeker(data []byte) *bytesReadSeeker {
	return &bytesReadSeeker{
		data: data,
	}
}

func (b *bytesReadSeeker) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}

	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *bytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int
	switch whence {
	case io.SeekStart:
		newPos = int(offset)
	case io.SeekCurrent:
		newPos = b.pos + int(offset)
	case io.SeekEnd:
		newPos = len(b.data) + int(offset)
	}

	if newPos < 0 {
		return 0, io.ErrUnexpectedEOF
	}

	b.pos = newPos
	return int64(b.pos), nil
}
