package telnetlib

import (
	"fmt"
	"io"
	"log"
	"net"
)

type DataHandlerFunc func(w io.Writer, r io.Reader)
type CmdHandlerFunc func(w io.Writer, r io.Reader)

var defaultDataHandlerFunc = func(w io.Writer, r io.Reader) {
	// Keep reading and do nothing
	for {
		b := make([]byte, 512)
		if _, err := r.Read(b); err != nil {
			return
		}
	}
}

var defaultCmdHandlerFunc = func(w io.Writer, r io.Reader) {
	// Keep reading and do nothing
	log.Printf("running default handler")
	for {
		log.Printf("reading")
		b := make([]byte, 512)
		_, err := r.Read(b)
		if err != nil {
			return
		}
	}
}

type TelnetOpts struct {
	Addr        string
	ServerOpts  []byte
	ClientOpts  []byte
	DataHandler DataHandlerFunc
	CmdHandler  CmdHandlerFunc
}

type TelnetServer struct {
	ServerOptions map[byte]bool
	ClientOptions map[byte]bool
	DataHandler   func(w io.Writer, r io.Reader)
	CmdHandler    func(w io.Writer, r io.Reader)
	ln            net.Listener
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
	if ts.DataHandler == nil {
		ts.DataHandler = defaultDataHandlerFunc
	}

	ts.CmdHandler = opts.CmdHandler
	if ts.CmdHandler == nil {
		ts.CmdHandler = defaultCmdHandlerFunc
	}
	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		panic(fmt.Sprintf("cannot start telnet server: %v", err))
	}
	ts.ln = ln
	return ts
}

// Accept accepts a connection and returns the Telnet connection
func (ts *TelnetServer) Accept() (*TelnetConn, error) {
	conn, _ := ts.ln.Accept()
	log.Printf("connection received")
	opts := connOpts{
		conn:       conn,
		cmdHandler: ts.CmdHandler,
		serverOpts: ts.ServerOptions,
		clientOpts: ts.ClientOptions,
		fsm:        newTelnetFSM(),
	}
	tc := newTelnetConn(opts)
	go tc.connectionLoop()
	go tc.readLoop()
	go ts.DataHandler(tc.handlerWriter, tc.dataRW)
	go tc.fsm.start()
	go tc.startNegotiation()
	return tc, nil
}
