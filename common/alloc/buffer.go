// Package alloc provides a light-weight memory allocation mechanism.
package alloc

import (
	"hash"
	"io"

	"v2ray.com/core/common/serial"
)

const (
	defaultOffset = 16
)

// Buffer is a recyclable allocation of a byte array. Buffer.Release() recycles
// the buffer into an internal buffer pool, in order to recreate a buffer more
// quickly.
type Buffer struct {
	head   []byte
	pool   Pool
	Value  []byte
	offset int
}

func CreateBuffer(container []byte, parent Pool) *Buffer {
	b := new(Buffer)
	b.head = container
	b.pool = parent
	b.Value = b.head[defaultOffset:]
	b.offset = defaultOffset
	return b
}

// Release recycles the buffer into an internal buffer pool.
func (b *Buffer) Release() {
	if b == nil || b.head == nil {
		return
	}
	if b.pool != nil {
		b.pool.Free(b)
	}
	b.head = nil
	b.Value = nil
	b.pool = nil
}

// Clear clears the content of the buffer, results an empty buffer with
// Len() = 0.
func (b *Buffer) Clear() *Buffer {
	b.offset = defaultOffset
	b.Value = b.head[b.offset:b.offset]
	return b
}

// Reset resets this Buffer into its original state.
func (b *Buffer) Reset() *Buffer {
	b.offset = defaultOffset
	b.Value = b.head
	return b
}

// AppendBytes appends one or more bytes to the end of the buffer.
func (b *Buffer) AppendBytes(bytes ...byte) *Buffer {
	b.Value = append(b.Value, bytes...)
	return b
}

// Append appends a byte array to the end of the buffer.
func (b *Buffer) Append(data []byte) *Buffer {
	b.Value = append(b.Value, data...)
	return b
}

// AppendString appends a given string to the end of the buffer.
func (b *Buffer) AppendString(s string) *Buffer {
	b.Value = append(b.Value, s...)
	return b
}

func (b *Buffer) AppendUint16(v uint16) *Buffer {
	b.Value = serial.Uint16ToBytes(v, b.Value)
	return b
}

func (b *Buffer) AppendUint32(v uint32) *Buffer {
	b.Value = serial.Uint32ToBytes(v, b.Value)
	return b
}

// Prepend prepends bytes in front of the buffer. Caller must ensure total bytes prepended is
// no more than 16 bytes.
func (b *Buffer) Prepend(data []byte) *Buffer {
	b.SliceBack(len(data))
	copy(b.Value, data)
	return b
}

func (b *Buffer) PrependBytes(data ...byte) *Buffer {
	return b.Prepend(data)
}

func (b *Buffer) PrependUint16(v uint16) *Buffer {
	b.SliceBack(2)
	serial.Uint16ToBytes(v, b.Value[:0])
	return b
}

func (b *Buffer) PrependUint32(v uint32) *Buffer {
	b.SliceBack(4)
	serial.Uint32ToBytes(v, b.Value[:0])
	return b
}

func (b *Buffer) PrependHash(h hash.Hash) *Buffer {
	b.SliceBack(h.Size())
	h.Sum(b.Value[:0])
	return b
}

// Bytes returns the content bytes of this Buffer.
func (b *Buffer) Bytes() []byte {
	return b.Value
}

// Slice cuts the buffer at the given position.
func (b *Buffer) Slice(from, to int) *Buffer {
	b.offset += from
	b.Value = b.Value[from:to]
	return b
}

// SliceFrom cuts the buffer at the given position.
func (b *Buffer) SliceFrom(from int) *Buffer {
	b.offset += from
	b.Value = b.Value[from:]
	return b
}

// SliceBack extends the Buffer to its front by offset bytes.
// Caller must ensure cumulated offset is no more than 16.
func (b *Buffer) SliceBack(offset int) *Buffer {
	newoffset := b.offset - offset
	if newoffset < 0 {
		panic("Negative buffer offset.")
	}
	b.Value = b.head[newoffset : b.offset+len(b.Value)]
	b.offset = newoffset
	return b
}

// Len returns the length of the buffer content.
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.Value)
}

func (b *Buffer) IsEmpty() bool {
	return b.Len() == 0
}

// IsFull returns true if the buffer has no more room to grow.
func (b *Buffer) IsFull() bool {
	return len(b.Value) == cap(b.Value)
}

// Write implements Write method in io.Writer.
func (b *Buffer) Write(data []byte) (int, error) {
	b.Append(data)
	return len(data), nil
}

// Read implements io.Reader.Read().
func (b *Buffer) Read(data []byte) (int, error) {
	if b.Len() == 0 {
		return 0, io.EOF
	}
	nBytes := copy(data, b.Value)
	if nBytes == b.Len() {
		b.Clear()
	} else {
		b.Value = b.Value[nBytes:]
		b.offset += nBytes
	}
	return nBytes, nil
}

func (b *Buffer) FillFrom(reader io.Reader) (int, error) {
	begin := b.Len()
	b.Value = b.Value[:cap(b.Value)]
	nBytes, err := reader.Read(b.Value[begin:])
	if err == nil {
		b.Value = b.Value[:begin+nBytes]
	}
	return nBytes, err
}

func (b *Buffer) String() string {
	return string(b.Value)
}

// NewSmallBuffer creates a Buffer with 1K bytes of arbitrary content.
func NewSmallBuffer() *Buffer {
	return smallPool.Allocate()
}

// NewBuffer creates a Buffer with 8K bytes of arbitrary content.
func NewBuffer() *Buffer {
	return mediumPool.Allocate()
}

// NewLargeBuffer creates a Buffer with 64K bytes of arbitrary content.
func NewLargeBuffer() *Buffer {
	return largePool.Allocate()
}

func NewBufferWithSize(size int) *Buffer {
	if size <= SmallBufferSize {
		return NewSmallBuffer()
	}

	if size <= BufferSize {
		return NewBuffer()
	}

	return NewLargeBuffer()
}

func NewLocalBuffer(size int) *Buffer {
	return CreateBuffer(make([]byte, size), nil)
}
