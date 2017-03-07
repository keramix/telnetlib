package telnetlib

import (
	"bytes"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

type cmd struct {
	cmdBuf []byte
	called bool
}

func (d *cmd) mockCmdHandler(w io.Writer, b []byte, tc *TelnetConn) {
	d.called = true
	d.cmdBuf = b
}

type opt struct {
	cmd    byte
	optn   byte
	called bool
}

func (o *opt) optCallback(cmd, option byte) {
	o.cmd = cmd
	o.optn = option
	o.called = true
}

type testSample struct {
	inputSeq []byte
	expState []state
	expOpt   []*opt
	expCmd   []*cmd
}

var samples = []testSample{
	testSample{
		inputSeq: []byte{10, 20, 5, 12, 34, 125, 98},
		expState: []state{0, 0, 0, 0, 0, 0, 0},
		expOpt:   []*opt{nil, nil, nil, nil, nil, nil, nil},
		expCmd:   []*cmd{nil, nil, nil, nil, nil, nil, nil},
	},
	testSample{
		inputSeq: []byte{Iac, Do, Echo, 10, 20, Iac, Will, Sga},
		expState: []state{cmdState, optionNegotiationState, dataState, dataState, dataState, cmdState, optionNegotiationState, dataState},
		expOpt:   []*opt{nil, nil, &opt{Do, Echo, true}, nil, nil, nil, nil, &opt{Will, Sga, true}},
		expCmd:   []*cmd{nil, nil, nil, nil, nil, nil, nil, nil},
	},
	testSample{
		inputSeq: []byte{10, 20, Iac, Ayt, 5, Iac, Ao},
		expState: []state{dataState, dataState, cmdState, dataState, dataState, cmdState, dataState},
		expOpt:   []*opt{nil, nil, nil, nil, nil, nil, nil},
		expCmd:   []*cmd{nil, nil, nil, &cmd{[]byte{Ayt}, true}, nil, nil, &cmd{[]byte{Ao}, true}},
	},
	testSample{
		inputSeq: []byte{10, Iac, Sb, 5, 12, Iac, Se},
		expState: []state{dataState, cmdState, subnegState, subnegState, subnegState, subnegEndState, dataState},
		expOpt:   []*opt{nil, nil, nil, nil, nil, nil, nil},
		expCmd:   []*cmd{nil, nil, nil, nil, nil, nil, &cmd{[]byte{Sb, 5, 12, Se}, true}},
	},
}

func TestFSM(t *testing.T) {
	for count, s := range samples {
		log.Printf("test sample %d", count)
		b := make([]byte, 512)
		outBuf := bytes.NewBuffer(b)
		inBuf := bytes.NewBuffer(s.inputSeq)
		cmdPtr := &cmd{
			called: false,
		}
		optPtr := &opt{
			called: false,
		}
		dummyConn := newMockConn(inBuf, outBuf)
		fsm := newTelnetFSM()
		opts := connOpts{
			conn:        dummyConn,
			fsm:         fsm,
			cmdHandler:  cmdPtr.mockCmdHandler,
			dataHandler: defaultDataHandlerFunc,
			optCallback: optPtr.optCallback,
		}
		tc := newTelnetConn(opts)
		go tc.dataHandlerWrapper(tc.handlerWriter, tc.dataRW)
		assert.Equal(t, fsm.curState, dataState)

		for i, ch := range s.inputSeq {
			ns := fsm.nextState(ch)
			fsm.curState = ns
			assert.Equal(t, s.expState[i], ns)
			if optPtr.called {
				assert.Equal(t, s.expOpt[i].cmd, optPtr.cmd)
				assert.Equal(t, s.expOpt[i].optn, optPtr.optn)
			} else {
				assert.Nil(t, s.expOpt[i])
			}
			if cmdPtr.called {
				exp := s.expCmd[i].cmdBuf
				actual := cmdPtr.cmdBuf
				assert.Equal(t, exp, actual)
			} else {
				assert.Nil(t, s.expCmd[i])
			}
			optPtr.called = false
			cmdPtr.called = false
		}
	}
}
