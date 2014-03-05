package main

import (
	"bufio"
	"io"
	"strconv"
)

func GitChunkReader(r io.Reader) io.Reader {
	bufr := bufio.NewReaderSize(r, 1024)
	header, err := bufr.Peek(5)
	if err != nil {
		panic(err)
	}

	if !is_chunked(header) {
		return bufr
	}

	return &git_chunk_reader{bufr, 0}
}

func is_chunked(header []byte) bool {
	return is_hex(header[0]) && is_hex(header[1]) && is_hex(header[2]) && is_hex(header[3]) && header[4] == '\n'
}

func is_hex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f'
}

type git_chunk_reader struct {
	bufr *bufio.Reader
	left int
}

func (r *git_chunk_reader) Read(b []byte) (int, error) {
	// read header
	if r.left == 0 {
		var (
			header [5]byte
		)
		n, err := io.ReadFull(r.bufr, header[:])
		if n == 1 && header[0] == '0' {
			return 0, io.EOF
		}
		if n != 5 {
			return 0, io.ErrUnexpectedEOF
		}

		i64, err := strconv.ParseInt(string(header[:4]), 16, 32)
		if err != nil {
			return 0, err
		}

		r.left = int(i64)
		return r.Read(b)
	}

	if len(b) > r.left {
		b = b[:r.left]
	}

	n, err := io.ReadFull(r.bufr, b)
	r.left -= n
	if err != nil {
		return n, err
	}

	if r.left == 0 {
		c, err := r.bufr.ReadByte()
		if err != nil {
			return n, err
		}

		if c != '\n' {
			return n, io.ErrUnexpectedEOF
		}
	}

	return n, nil
}
