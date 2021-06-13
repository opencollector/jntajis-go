package jntajis

import "io"

type Encoder interface {
	Encode(m string) ([]byte, error)
	Bytes(b []byte) ([]byte, error)
	String(s string) (string, error)
	Writer(w io.Writer) io.Writer
}

type Decoder interface {
	Decode([]byte) (string, error)
	Bytes(b []byte) ([]byte, error)
	String(s string) (string, error)
	Reader(r io.Reader) io.Reader
}
