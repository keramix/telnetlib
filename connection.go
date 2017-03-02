package telnetlib

import (
	"bytes"
	"io"
	"io/ioutil"
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
	serverOpts         map[byte]bool
	clientOpts         map[byte]bool
	dataRW             io.ReadWriter
	cmdBuffer          bytes.Buffer
	fsm                *telnetFSM
	fsmInputCh         chan byte
	handlerWriter      io.Writer
	cmdHandler         CmdHandlerFunc
	dataHandler        DataHandlerFunc
	dataHandlerCloseCh chan chan struct{}
	dataWrittenCh      chan bool
	optionCallback     func(byte, byte)
	connReadDoneCh     chan chan struct{}
	connWriteDoneCh    chan chan struct{}
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
		cmdHandler:         opts.cmdHandler,
		dataHandler:        opts.dataHandler,
		dataHandlerCloseCh: make(chan chan struct{}),
		dataWrittenCh:      make(chan bool),
		serverOpts:         opts.serverOpts,
		clientOpts:         opts.clientOpts,
		optionCallback:     opts.optCallback,
		fsmInputCh:         make(chan byte),
		connReadDoneCh:     make(chan chan struct{}),
		connWriteDoneCh:    make(chan chan struct{}),
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
	defer func() {
		log.Printf("read loop closed")
	}()
	for {
		buf := make([]byte, 4096)
		n, err := c.conn.Read(buf)
		if n > 0 {
			log.Printf("read %d bytes from the TCP Connection %v", n, buf[:n])
			c.readCh <- buf[:n]
		}
		if err != nil {
			log.Printf("connection read: %v", err)
			c.Close()
			break
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
	c.closeConnLoopRead()
	c.closeConnLoopWrite()
	c.closeFSM()
	c.closeDatahandler()
	log.Printf("telnet connection closed")
}

func (c *TelnetConn) closeConnLoopRead() {
	connLoopReadCh := make(chan struct{})
	c.connReadDoneCh <- connLoopReadCh
	<-connLoopReadCh
	log.Printf("connection loop read-side closed")
}

func (c *TelnetConn) closeConnLoopWrite() {
	connLoopWriteCh := make(chan struct{})
	c.connWriteDoneCh <- connLoopWriteCh
	<-connLoopWriteCh
	log.Printf("connection loop write-side closed")
}

func (c *TelnetConn) closeFSM() {
	fsmCh := make(chan struct{})
	c.fsm.doneCh <- fsmCh
	<-fsmCh
}

func (c *TelnetConn) closeDatahandler() {
	dataCh := make(chan struct{})
	c.dataHandlerCloseCh <- dataCh
	<-dataCh
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
	defer func() {
		log.Printf("data handler closed")
	}()
	for {
		select {
		case ch := <-c.dataHandlerCloseCh:
			ch <- struct{}{}
			log.Printf("before break")
			return
		case <-c.dataWrittenCh:
			if b, err := ioutil.ReadAll(r); err == nil {
				log.Printf("read %d bytes", len(b))
				c.dataHandler(w, b)
			}
		}
	}
}

func (c *TelnetConn) cmdHandlerWrapper(w io.Writer, r io.Reader) {
	if cmd, err := ioutil.ReadAll(r); err == nil {
		c.cmdHandler(w, cmd)
	}
}
