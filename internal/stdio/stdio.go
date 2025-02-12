package stdio

import (
	"io"
)

func New(reader io.ReadCloser, writer io.WriteCloser) *ReadWriteCloser {
	return &ReadWriteCloser{reader: reader, writer: writer}
}

type ReadWriteCloser struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (r ReadWriteCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r ReadWriteCloser) Write(p []byte) (int, error) {
	return r.writer.Write(p)
}

func (r ReadWriteCloser) Close() error {
	if err := r.reader.Close(); err != nil {
		_ = r.writer.Close()
		return err
	}
	return r.writer.Close()
}
