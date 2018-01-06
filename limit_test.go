package uhttp

import (
	"bytes"
	"io"
	"testing"
)

func TestLimitedWriter(t *testing.T) {
	var buf bytes.Buffer
	var n int
	var err error

	lw := &limitedWriter{Writer: &buf, N: 5}

	n, err = lw.Write(nil)
	if n != 0 || err != nil {
		t.Errorf("nil write should return 0/nil, got %d/%v", n, err)
	}

	n, err = lw.Write([]byte{})
	if n != 0 || err != nil {
		t.Errorf("empty write should return 0/nil, got %d/%v", n, err)
	}

	n, err = lw.Write([]byte("12345"))
	if n != 5 || err != nil {
		t.Errorf("5-byte write into capacity 5 writer should return 5/nil, got %d/%v", n, err)
	}

	n, err = lw.Write([]byte("678"))
	if n != 0 || err != io.ErrShortWrite {
		t.Errorf("3-byte write after hitting 5 cap should return 0/io.ErrShortWrite, got %d/%v", n, err)
	}

	if string(buf.Bytes()) != "12345" {
		t.Errorf("should have written %q to underlying buffer, got %q", "12345", buf.Bytes())
	}

	buf.Reset()
	lw = &limitedWriter{Writer: &buf, N: 5}

	n, err = lw.Write([]byte("abcdefg"))
	if n != 5 || err != io.ErrShortWrite {
		t.Errorf("6-byte write into capacity 5 writer should return 5/io.ErrShortWrite, got %d/%v", n, err)
	}

	if string(buf.Bytes()) != "abcde" {
		t.Errorf("should have written %q to underlying buffer, got %q", "abcde", buf.Bytes())
	}
}
