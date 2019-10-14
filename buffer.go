// Package buffer provides a simple buffer that overflows onto disk.
package buffer

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
)

// Buffer is a memory buffer with a disk overflow.
//
// The zero value of Buffer is a pure disk buffer and is safe to use without
// any initialization.
type Buffer struct {
	max int

	head bytes.Buffer
	file *os.File

	freadOffset    int64
	fwriteOffset   int64
	fcurrentOffset int64
}

// New creates a buffer with a max number of bytes in memory.
func New(memory int) *Buffer {
	return &Buffer{
		max: memory,
	}
}

func (b *Buffer) Read(p []byte) (_ int, err error) {
	if b.head.Len() > 0 {
		n, err := b.head.Read(p)
		if err == io.EOF && b.file != nil {
			return n, io.EOF
		}

		return n, err
	}

	if b.fcurrentOffset != b.freadOffset {
		if b.fcurrentOffset, err = b.file.Seek(b.freadOffset, 0); err != nil {
			return 0, err
		}
	}

	n, err := b.file.Read(p)
	if err != nil {
		return 0, err
	}

	b.freadOffset += int64(n)
	b.fcurrentOffset = b.freadOffset

	return n, nil
}

func (b *Buffer) Write(p []byte) (_ int, err error) {
	if b.file != nil {
		if b.fcurrentOffset != b.fwriteOffset {
			if b.fcurrentOffset, err = b.file.Seek(b.fwriteOffset, 0); err != nil {
				return 0, err
			}
		}

		n, err := b.file.Write(p)
		if err != nil {
			return 0, err
		}

		b.fwriteOffset += int64(n)
		b.fcurrentOffset = b.fwriteOffset

		return n, nil
	}

	if b.head.Len()+len(p) <= b.max {
		return b.head.Write(p)
	}

	boundary := b.max - b.head.Len()
	n1, _ := b.head.Write(p[:boundary])

	b.file, err = ioutil.TempFile("", "")
	if err != nil {
		return 0, err
	}

	n2, err := b.file.Write(p[boundary:])
	if err != nil {
		return 0, err
	}

	b.fwriteOffset = int64(n2)
	b.fcurrentOffset = b.fwriteOffset

	return n1 + n2, nil
}

// Close resets the buffer state, removing the buffer file from disk if there
// is any.
//
// The buffer may be reused as new after Close is called.
func (b *Buffer) Close() error {
	b.head.Truncate(0)

	if b.file == nil {
		return nil
	}

	if err := os.Remove(b.file.Name()); err != nil {
		return err
	}

	b.file = nil
	b.freadOffset = 0
	b.fwriteOffset = 0
	b.fcurrentOffset = 0

	return nil
}
