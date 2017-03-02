package telnetlib

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type connOpts struct {
	conn        net.Conn
	fsm         *telnetFSM
	cmdHandler  CmdHandlerFunc
	dataHandler DataHandlerFunc
	serverOpts  map[byte]bool
	clientOpts  map[byte]bool
	optCallback func(byte, byte)
}

type TelnetConn struct {
	conn              net.Conn
	readCh            chan []byte
	writeCh           chan []byte
	acceptedOpts      map[byte]bool
	unackedServerOpts map[byte]bool
	unackedClientOpts map[byte]bool
	//server            *TelnetServer
	serverOpts      map[byte]bool
	clientOpts      map[byte]bool
	dataRW          io.ReadWriter
	cmdBuffer       bytes.Buffer
	fsm             *telnetFSM
	fsmInputCh      chan byte
	handlerWriter   io.Writer
	cmdHandler      CmdHandlerFunc
	dataHandler     DataHandlerFunc
	optionCallback  func(byte, byte)
	readDoneCh      chan chan struct{}
	connReadDoneCh  chan chan struct{}
	connWriteDoneCh chan chan struct{}
}

// Safely read/write concurrently to the data Buffer
// databuffer is written to by the FSM and it is read from by the dataHandler
type dataReadWriter struct {
	dataBuffer bytes.Buffer
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

func newTelnetConn(opts connOpts) *TelnetConn {
	tc := &TelnetConn{
		conn:              opts.conn,
		readCh:            make(chan []byte),
		writeCh:           make(chan []byte),
		acceptedOpts:      make(map[byte]bool),
		unackedServerOpts: make(map[byte]bool),
		unackedClientOpts: make(map[byte]bool),
		//server:            telnetServer,
		cmdHandler:      opts.cmdHandler,
		dataHandler:     opts.dataHandler,
		serverOpts:      opts.serverOpts,
		clientOpts:      opts.clientOpts,
		optionCallback:  opts.optCallback,
		fsmInputCh:      make(chan byte),
		readDoneCh:      make(chan chan struct{}),
		connReadDoneCh:  make(chan chan struct{}),
		connWriteDoneCh: make(chan chan struct{}),
	}
	if tc.optionCallback == nil {
		tc.optionCallback = tc.handleOptionCommand
	}
	tc.handlerWriter = &connectionWriter{
		ch: tc.writeCh,
	}
	tc.dataRW = &dataReadWriter{
		dataMux: new(sync.Mutex),
	}
	fsm := opts.fsm
	fsm.tc = tc
	tc.fsm = fsm
	return tc
}

func (c *TelnetConn) connectionLoop() {
	log.Printf("Entered connectionLoop")
	// this is the reading thread
	go func() {
		for {
			select {
			case readBytes := <-c.readCh:
				for _, ch := range readBytes {
					//log.Printf("putting character %v on the fsm", ch)
					c.fsmInputCh <- ch
					//log.Printf("character already put on the fsm")
				}

			case ch := <-c.connReadDoneCh:
				ch <- struct{}{}
				return
			}
		}
	}()
	// this is the writing thread
	go func() {
		for {
			select {
			case writeBytes := <-c.writeCh:
				//log.Printf("writing to the connection")
				c.conn.Write(writeBytes)
			//log.Printf("connections already wrote")
			case ch := <-c.connWriteDoneCh:
				ch <- struct{}{}
				return
			}
		}
	}()
}

// reads from the connection and dumps into the connection read channel
func (c *TelnetConn) readLoop() {
	for {
		select {
		case ch := <-c.readDoneCh:
			ch <- struct{}{}
			return
		default:
			buf := make([]byte, 256)
			n, err := c.conn.Read(buf)
			if err != nil {
				//log.Printf("read error: %v", err)
			}
			if n > 0 {
				log.Printf("read %d bytes from the TCP Connection %v", n, buf[:n])
			}
			c.readCh <- buf[:n]
		}
	}
}

func (c *TelnetConn) startNegotiation() {
	for k := range c.serverOpts {
		log.Printf("sending WILL %d", k)
		c.unackedServerOpts[k] = true
		c.sendCmd(WILL, k)
	}
	for k := range c.clientOpts {
		log.Printf("sending DO %d", k)
		c.unackedClientOpts[k] = true
		c.sendCmd(DO, k)
	}
}

// Close closes the telnet connection
func (c *TelnetConn) Close() {
	log.Printf("Closing the connection")
	c.conn.Close()
	readLoopCh := make(chan struct{})
	connLoopReadCh := make(chan struct{})
	connLoopWriteCh := make(chan struct{})
	fsmCh := make(chan struct{})
	c.readDoneCh <- readLoopCh
	<-readLoopCh
	log.Printf("read loop closed")
	c.connReadDoneCh <- connLoopReadCh
	<-connLoopReadCh
	log.Printf("fsm loop closed")
	c.connWriteDoneCh <- connLoopWriteCh
	<-connLoopWriteCh
	log.Printf("write loop closed")
	c.fsm.doneCh <- fsmCh
	<-fsmCh
	log.Printf("fsm closed")
	log.Printf("telnet connection closed")
}

func (c *TelnetConn) sendCmd(cmd byte, opt byte) {
	b := []byte{IAC, cmd, opt}
	log.Printf("Sending command: %v %v", cmd, opt)
	c.writeCh <- b
	//log.Printf("command sent!")
}

func (c *TelnetConn) handleOptionCommand(cmd byte, opt byte) {
	if cmd == WILL || cmd == WONT {
		if _, ok := c.clientOpts[opt]; !ok {
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
		if _, ok := c.serverOpts[opt]; !ok {
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

func (c *TelnetConn) dataHandlerWrapper(w io.Writer, r io.Reader) {
	for {
		buf := make([]byte, 512)
		n, _ := r.Read(buf)
		if n > 0 {
			log.Printf("read %d bytes", n)
			fmt.Printf("%v", w)
			fmt.Printf("%v", buf[:n])
			c.dataHandler(w, buf[:n])
		}
	}
}

func (c *TelnetConn) cmdHandlerWrapper(w io.Writer, r io.Reader) {
	var cmd []byte
	for {
		buf := make([]byte, 512)
		n, err := r.Read(buf)
		if n > 0 {
			cmd = append(cmd, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("unexpected error: %v", err)
			break
		}
	}
	c.cmdHandler(w, cmd)
}
