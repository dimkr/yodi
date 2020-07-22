package mqtt

import (
	"encoding/binary"
	"errors"
	"io"
)

type StringReader struct {
	io.Reader
}

type StringWriter struct {
	io.Writer
}

func (w *StringWriter) Write(p []byte) (int, error) {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(p)))

	n, err := w.Writer.Write(length)
	if err != nil {
		return n, err
	}

	return w.Writer.Write(p)
}

func (r *StringReader) Read(p []byte) (int, error) {
	buf := make([]byte, 2)
	n, err := r.Reader.Read(buf)
	if err != nil {
		return n, err
	}
	if n != 2 {
		return 0, errors.New("failed to read length")
	}

	length := int(binary.BigEndian.Uint16(buf))
	if len(p) < length {
		return 0, errors.New("buffer is too small")
	}

	return r.Reader.Read(p[:length])
}
