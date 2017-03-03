package telnetlib

import (
	"bytes"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		inputSeq: []byte{IAC, DO, ECHO, 10, 20, IAC, WILL, SGA},
		expState: []state{cmdState, optionNegotiationState, dataState, dataState, dataState, cmdState, optionNegotiationState, dataState},
		expOpt:   []*opt{nil, nil, &opt{DO, ECHO, true}, nil, nil, nil, nil, &opt{WILL, SGA, true}},
		expCmd:   []*cmd{nil, nil, nil, nil, nil, nil, nil, nil},
	},
	testSample{
		inputSeq: []byte{10, 20, IAC, AYT, 5, IAC, AO},
		expState: []state{dataState, dataState, cmdState, dataState, dataState, cmdState, dataState},
		expOpt:   []*opt{nil, nil, nil, nil, nil, nil, nil},
		expCmd:   []*cmd{nil, nil, nil, &cmd{[]byte{AYT}, true}, nil, nil, &cmd{[]byte{AO}, true}},
	},
	testSample{
		inputSeq: []byte{10, IAC, SB, 5, 12, IAC, SE},
		expState: []state{dataState, cmdState, subnegState, subnegState, subnegState, subnegEndState, dataState},
		expOpt:   []*opt{nil, nil, nil, nil, nil, nil, nil},
		expCmd:   []*cmd{nil, nil, nil, nil, nil, nil, &cmd{[]byte{SB, 5, 12, SE}, true}},
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
