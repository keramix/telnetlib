package telnetlib

import (
	"io"
	"net"
	"time"
)

type mockListener struct {
	conn net.Conn
}

func (l *mockListener) Accept() (net.Conn, error) {
	return l.conn, nil
}

func (l *mockListener) Close() error {
	return nil
}

func (l *mockListener) Addr() net.Addr {
	return nil
}

type mockConn struct {
	r io.Reader
	w io.Writer
}

func (c *mockConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *mockConn) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) LocalAddr() net.Addr {
	return nil
}

func (c *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (c *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func newMockConn(r io.Reader, w io.Writer) net.Conn {
	return &mockConn{
		r: r,
		w: w,
	}
}

func newMockListener(conn net.Conn) net.Listener {
	return &mockListener{
		conn: conn,
	}
}
