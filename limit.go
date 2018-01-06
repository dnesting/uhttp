package uhttp

import (
	"io"
	"sync"
)

type limitedWriter struct {
	io.Writer
	mu sync.Mutex
	N  int
}

func (w *limitedWriter) Write(b []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var short bool
	if len(b) > w.N {
		b = b[:w.N]
		short = true
	}
	n, err = w.Writer.Write(b)
	if err == nil && short {
		err = io.ErrShortWrite
	}
	w.N -= n
	return
}
