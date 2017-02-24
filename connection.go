package telnetlib

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"
)

type telnetConn struct {
	conn              net.Conn
	readCh            chan []byte
	writeCh           chan []byte
	acceptedOpts      map[byte]bool
	unackedServerOpts map[byte]bool
	unackedClientOpts map[byte]bool
	server            *TelnetServer
	dataRW            io.ReadWriter
	cmdBufMutex       *sync.Mutex
	cmdBuffer         bytes.Buffer
	fsm               *telnetFSM
	fsmInputCh        chan byte
	handlerWriter     io.Writer
}

// Safely read/write concurrently to the data Buffer
// databuffer is written to by the FSM and it is read from by the dataHandler
type dataReadWriter struct {
	dataBuffer *bytes.Buffer
	dataMux    *sync.Mutex
}

func (drw *dataReadWriter) Read(p []byte) (int, error) {
	drw.dataMux.Lock()
	defer drw.dataMux.Unlock()
	return drw.dataBuffer.Read(p)
}

func (drw *dataReadWriter) Write(p []byte) (int, error) {
	drw.dataMux.Lock()
	defer drw.dataMux.Unlock()
	return drw.dataBuffer.Write(p)
}

type connectionWriter struct {
	ch chan []byte
}

func (cw *connectionWriter) Write(b []byte) (int, error) {
	cw.ch <- b
	return len(b), nil
}

func newTelnetConn(conn net.Conn, telnetServer *TelnetServer) *telnetConn {
	tc := &telnetConn{
		conn:              conn,
		readCh:            make(chan []byte),
		writeCh:           make(chan []byte),
		acceptedOpts:      make(map[byte]bool),
		unackedServerOpts: make(map[byte]bool),
		unackedClientOpts: make(map[byte]bool),
		server:            telnetServer,
		fsmInputCh:        make(chan byte),
	}
	tc.handlerWriter = &connectionWriter{
		ch: tc.writeCh,
	}
	tc.dataRW = &dataReadWriter{
		dataBuffer: bytes.NewBuffer(make([]byte, 512)),
		dataMux:    new(sync.Mutex),
	}
	tc.fsm = newTelnetFSM(tc)
	go tc.connectionLoop()
	go tc.readLoop()
	go tc.server.DataHandler(tc.handlerWriter, tc.dataRW)
	tc.startNegotiation()
	return tc
}

func (c *telnetConn) connectionLoop() {
	log.Printf("Entered connectionLoop")
	for {
		select {
		case readBytes := <-c.readCh:
			// write the read bytes byte by byte to the fsm input channel
			for _, ch := range readBytes {
				c.fsmInputCh <- ch
			}
		case writeBytes := <-c.writeCh:
			//log.Printf("writing: %v", writeBytes)
			c.conn.Write(writeBytes)
		}
	}
}

// reads from the connection and dumps into the connection read channel
func (c *telnetConn) readLoop() {
	for {
		buf := make([]byte, 256)
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Printf("read error: %v", err)
		}
		log.Printf("read %d bytes from the TCP Connection %v", n, buf[:n])
		c.readCh <- buf[:n]
		log.Printf("wrote to read channel")
	}
}

func (c *telnetConn) startNegotiation() {
	for k := range c.server.ServerOptions {
		log.Printf("sending WILL %d", k)
		c.unackedServerOpts[k] = true
		c.sendCmd(WILL, k)
	}
	for k := range c.server.ClientOptions {
		log.Printf("sending DO %d", k)
		c.unackedClientOpts[k] = true
		c.sendCmd(DO, k)
	}
}

func (c *telnetConn) sendCmd(cmd byte, opt byte) {
	b := []byte{IAC, cmd, opt}
	c.writeCh <- b
}

func (c *telnetConn) handleOptionCommand(cmd byte, opt byte) {
	ts := c.server
	if cmd == WILL || cmd == WONT {
		if _, ok := ts.ClientOptions[opt]; !ok {
			c.sendCmd(DONT, opt)
			return
		}
		if cmd == WONT {
			delete(c.acceptedOpts, opt)
		}
		// if this is a reply to a DO
		if _, ok := c.unackedClientOpts[opt]; ok {
			// remove it from the unackedClientOpts
			delete(c.unackedClientOpts, opt)
		} else {
			c.sendCmd(DO, opt)
		}
	}

	if cmd == DO || cmd == DONT {
		if _, ok := ts.ServerOptions[opt]; !ok {
			c.sendCmd(WONT, opt)
			return
		}
		if cmd == DONT {
			delete(c.acceptedOpts, opt)
		}
		// if this is a reply to a DO
		if _, ok := c.unackedServerOpts[opt]; ok {
			log.Printf("removing from the unack list")
			// remove it from the unackedClientOpts
			delete(c.unackedServerOpts, opt)
		} else {
			log.Printf("Sending WILL command")
			c.sendCmd(WILL, opt)
		}
	}

	log.Printf("finished handling Option command")
}
