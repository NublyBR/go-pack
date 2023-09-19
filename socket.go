package pack

import (
	"bufio"
	"bytes"
	"net"
	"sync"
	"time"
)

type Socket interface {
	// Read object from socket
	Read() (any, error)

	// Write object to socket
	Write(any) error

	// Read object from socket with a timeout
	ReadTimeout(timeout time.Duration) (any, error)

	// Write object to socket with a timeout
	WriteTimeout(data any, timeout time.Duration) error

	// Close socket
	Close() error

	// Get total bytes read
	BytesRead() uint64

	// Get total bytes written
	BytesWritten() uint64

	// Reset total bytes read
	ResetRead()

	// Reset total bytes written
	ResetWritten()

	// Deallocate write buffer to free memory
	ZeroBuffer()
}

type socket struct {
	conn net.Conn

	writeBuffer    *bytes.Buffer
	bufferedReader *bufio.Reader

	unpacker Unpacker
	packer   Packer

	wlock sync.Mutex
	rlock sync.Mutex

	// For the socket implementation, bytes written may differ from
	// packer.BytesWritten(), since if packer.Encode() errors, no bytes
	// will be written to the socket.
	written uint64
}

func NewSocket(conn net.Conn, options Options) Socket {
	if options.WithObjects == nil {
		panic("WithObjects may not be nil in Socket")
	}

	var (
		writeBuffer    = bytes.NewBuffer(nil)
		bufferedReader = bufio.NewReader(conn)
	)

	return &socket{
		conn: conn,

		writeBuffer:    writeBuffer,
		bufferedReader: bufferedReader,

		unpacker: NewUnpacker(bufferedReader, options),
		packer:   NewPacker(writeBuffer, options),
	}
}

func (s *socket) read() (any, error) {
	var receiver any

	if err := s.unpacker.Decode(&receiver); err != nil {
		return nil, err
	}

	return receiver, nil
}

func (s *socket) write(data any) error {
	s.writeBuffer.Reset()

	if err := s.packer.Encode(data); err != nil {
		return err
	}

	n, err := s.conn.Write(s.writeBuffer.Bytes())

	s.written += uint64(n)

	return err
}

func (s *socket) Read() (any, error) {
	s.rlock.Lock()
	defer s.rlock.Unlock()

	if err := s.conn.SetReadDeadline(time.Time{}); err != nil {
		return nil, err
	}

	return s.read()
}

func (s *socket) Write(data any) error {
	s.wlock.Lock()
	defer s.wlock.Unlock()

	if err := s.conn.SetWriteDeadline(time.Time{}); err != nil {
		return err
	}

	return s.write(data)
}

func (s *socket) ReadTimeout(timeout time.Duration) (any, error) {
	s.rlock.Lock()
	defer s.rlock.Unlock()

	if err := s.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	return s.read()
}

func (s *socket) WriteTimeout(data any, timeout time.Duration) error {
	s.wlock.Lock()
	defer s.wlock.Unlock()

	if err := s.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	return s.write(data)
}

func (s *socket) Close() error {
	return s.conn.Close()
}

func (s *socket) BytesRead() uint64 {
	s.rlock.Lock()
	defer s.rlock.Unlock()

	return s.unpacker.BytesRead()
}

func (s *socket) BytesWritten() uint64 {
	s.wlock.Lock()
	defer s.wlock.Unlock()

	return s.written
}

func (s *socket) ResetRead() {
	s.rlock.Lock()
	defer s.rlock.Unlock()

	s.unpacker.ResetCounter()
}

func (s *socket) ResetWritten() {
	s.wlock.Lock()
	defer s.wlock.Unlock()

	s.written = 0
	s.packer.ResetCounter()
}

func (s *socket) ZeroBuffer() {
	s.wlock.Lock()
	defer s.wlock.Unlock()

	s.writeBuffer = bytes.NewBuffer(nil)
}
