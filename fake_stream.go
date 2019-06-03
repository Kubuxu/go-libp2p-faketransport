package fk

import (
	"io"
	"time"
)

type fkStream struct {
	in  *io.PipeReader
	out *io.PipeWriter
}

func (fk *fkStream) Read(p []byte) (n int, err error) {
	return fk.in.Read(p)
}

func (fk *fkStream) Write(p []byte) (n int, err error) {
	return fk.out.Write(p)
}

func (fk *fkStream) Close() error {
	return fk.out.Close()
}

func (fk *fkStream) Reset() error {
	fk.out.Close()
	fk.in.Close()
	return nil
}

func (*fkStream) SetDeadline(time.Time) error {
	return nil
}

func (*fkStream) SetReadDeadline(time.Time) error {
	return nil
}

func (*fkStream) SetWriteDeadline(time.Time) error {
	return nil
}
