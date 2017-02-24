package telnetlib

import "log"

type state int

const (
	dataState state = iota
	optionNegotiationState
	cmdState
	subnegState
	subnegEndState
	errorState
)

type telnetFSM struct {
	curState state
	tc       *telnetConn
}

func newTelnetFSM(tc *telnetConn) *telnetFSM {
	f := &telnetFSM{
		curState: dataState,
		tc:       tc,
	}
	go f.start()
	return f
}

func (fsm *telnetFSM) start() {
	for {
		select {
		case ch := <-fsm.tc.fsmInputCh:
			//log.Printf("FSM state is %d", fsm.curState)
			fsm.nextState(ch)
		}
	}
}

// this function returns what the next state is and performs the appropriate action
func (fsm *telnetFSM) nextState(ch byte) {
	var nextState state
	b := []byte{ch}
	switch fsm.curState {
	case dataState:
		if ch != IAC {
			//log.Printf("FSM is writing %d to the dataRW", ch)
			fsm.tc.dataRW.Write(b)
			//log.Printf("FSM finished writing to the dataRW")
			nextState = dataState
		} else {
			nextState = cmdState
		}

	case cmdState:
		if ch == IAC { // this is an escaping of IAC to send it as data
			fsm.tc.dataRW.Write(b)
			nextState = dataState
		} else if ch == DO || ch == DONT || ch == WILL || ch == WONT {
			fsm.tc.cmdBuffer.WriteByte(ch)
			nextState = optionNegotiationState
		} else if ch == SB {
			fsm.tc.cmdBuffer.WriteByte(ch)
			fsm.tc.server.CmdHandler(fsm.tc.handlerWriter, &fsm.tc.cmdBuffer)
			fsm.tc.cmdBuffer.Reset()
			nextState = subnegState
		} else { // anything else
			fsm.tc.cmdBuffer.WriteByte(ch)
			nextState = dataState
		}
	case optionNegotiationState:
		fsm.tc.cmdBuffer.WriteByte(ch)
		opt := ch
		cmd := fsm.tc.cmdBuffer.Bytes()[0]
		fsm.tc.handleOptionCommand(cmd, opt)
		fsm.tc.cmdBuffer.Reset()
		nextState = dataState
	case subnegState:
		if ch == IAC {
			nextState = subnegEndState
		} else {
			nextState = subnegState
			fsm.tc.cmdBuffer.WriteByte(ch)
		}
	case subnegEndState:
		if ch == SE {
			fsm.tc.cmdBuffer.WriteByte(ch)
			fsm.tc.server.CmdHandler(fsm.tc.handlerWriter, &fsm.tc.cmdBuffer)
			fsm.tc.cmdBuffer.Reset()
			nextState = dataState
		} else {
			nextState = errorState
		}
	case errorState:
		nextState = dataState
		log.Printf("Finite state machine is in an error state. This should not happen for correct telnel protocol syntax")
	}
	fsm.curState = nextState
}
