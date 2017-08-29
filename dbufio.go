package dbufio

import (
	"fmt"
	"io"

	"github.com/xaevman/crash"
)

type Reader struct {
	backbufferIdx int
	buffers       [2][]byte
	bufferIdx     int
	bufferSize    int
	counts        [2]int
	errors        [2]error
	reader        io.Reader
	readIdx       int
	swapReady     chan bool
	fillReady     chan bool
}

func NewReader(reader io.Reader, size int) *Reader {
	r := &Reader{}
	r.reader = reader
	r.readIdx = 0
	r.bufferSize = size
	r.backbufferIdx = 1
	r.bufferIdx = 0
	r.swapReady = make(chan bool)
	r.fillReady = make(chan bool)

	for i := range r.buffers {
		r.buffers[i] = make([]byte, size)
	}

	c, err := r.reader.Read(r.buffers[r.bufferIdx])
	r.counts[r.bufferIdx] = c
	r.errors[r.bufferIdx] = err

	fmt.Printf("DoubleBufferedReader: %d bytes\n", c)

	go func() {
		defer crash.HandleAll()
		r.fillBackBuffer()
	}()
	r.swapReady <- true

	return r
}

func (r *Reader) Read(p []byte) (n int, err error) {
	copied := 0

	i := int64(0)
	for ; copied < len(p); i++ {
		// if we're at the end of our current read buffer, swap buffers to handle more reads
		if r.readIdx >= r.bufferSize || r.readIdx >= r.counts[r.bufferIdx] {
			r.swapBuffers()

			if r.counts[r.bufferIdx] < 1 {
				return r.counts[r.bufferIdx], r.errors[r.bufferIdx]
			}
		}

		// copy the calculated number of bytes
		count := copy(p[copied:], r.buffers[r.bufferIdx][r.readIdx:])

		// incrememnt our read buffer index by the copied number of bytes
		r.readIdx += count
		copied += count
	}

	return copied, nil
}

func (r *Reader) fillBackBuffer() {
	for {
		_, more := <-r.swapReady
		if !more {
			return
		}

		c, err := r.reader.Read(r.buffers[r.backbufferIdx])
		r.counts[r.backbufferIdx] = c
		r.errors[r.backbufferIdx] = err

		r.fillReady <- true
	}
}

func (r *Reader) swapBuffers() {
	<-r.fillReady

	if r.bufferIdx == 0 {
		r.bufferIdx = 1
		r.backbufferIdx = 0
	} else {
		r.bufferIdx = 0
		r.backbufferIdx = 1
	}

	r.readIdx = 0

	r.swapReady <- true
}

func min(v1, v2 int) int {
	if v1 < v2 {
		return v1
	}

	return v2
}
