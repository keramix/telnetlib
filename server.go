package telnetlib

import (
	"fmt"
	"io"
	"log"
	"net"
)

type DataHandlerFunc func(w io.Writer, r io.Reader)
type CmdHandlerFunc func(w io.Writer, r io.Reader)

type TelnetOpts struct {
	Addr        string
	ServerOpts  []byte
	ClientOpts  []byte
	DataHandler DataHandlerFunc
	CmdHandler  CmdHandlerFunc
}

type TelnetServer struct {
	ServerOptions  map[byte]bool
	ClientOptions  map[byte]bool
	optionCallback func(conn net.Conn, cmd byte, opt byte)
	DataHandler    func(w io.Writer, r io.Reader)
	CmdHandler     func(w io.Writer, r io.Reader)
	ln             net.Listener
}

func NewTelnetServer(opts TelnetOpts) *TelnetServer {
	ts := new(TelnetServer)
	ts.ClientOptions = make(map[byte]bool)
	ts.ServerOptions = make(map[byte]bool)
	for _, v := range opts.ServerOpts {
		ts.ServerOptions[v] = true
	}
	for _, v := range opts.ClientOpts {
		ts.ClientOptions[v] = true
	}
	ts.DataHandler = opts.DataHandler
	ts.CmdHandler = opts.CmdHandler
	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		panic(fmt.Sprintf("cannot start telnet server: %v", err))
	}
	ts.ln = ln
	return ts
}

func (ts *TelnetServer) Serve() {
	for {
		conn, _ := ts.ln.Accept()
		log.Printf("connection received")
		go newTelnetConn(conn, ts)
	}
}
