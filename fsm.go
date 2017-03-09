package telnetlib

import log "github.com/Sirupsen/logrus"

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
	tc       *TelnetConn
}

func newTelnetFSM() *telnetFSM {
	f := &telnetFSM{
		curState: dataState,
	}
	return f
}

func (fsm *telnetFSM) start() {
	defer func() {
		log.Infof("FSM closed")
	}()
	for {
		b := make([]byte, 4096)
		n, err := fsm.readFromRawConnection(b)
		if n > 0 {
			log.Debugf("read %d bytes from the TCP Connection %v", n, b[:n])
			for i := 0; i < n; i++ {
				ch := b[i]
				ns := fsm.nextState(ch)
				fsm.curState = ns
			}
		}
		if err != nil {
			log.Debugf("connection read: %v", err)
			fsm.tc.Close()
			break
		}
	}
}

func (fsm *telnetFSM) readFromRawConnection(b []byte) (int, error) {
	return fsm.tc.conn.Read(b)
}

// this function returns what the next state is and performs the appropriate action
func (fsm *telnetFSM) nextState(ch byte) state {
	var nextState state
	b := []byte{ch}
	switch fsm.curState {
	case dataState:
		if ch != Iac {
			fsm.tc.dataRW.Write(b)
			fsm.tc.dataWrittenCh <- true
			nextState = dataState
		} else {
			nextState = cmdState
		}

	case cmdState:
		if ch == Iac { // this is an escaping of IAC to send it as data
			fsm.tc.dataRW.Write(b)
			fsm.tc.dataWrittenCh <- true
			nextState = dataState
		} else if ch == Do || ch == Dont || ch == Will || ch == Wont {
			fsm.tc.cmdBuffer.WriteByte(ch)
			nextState = optionNegotiationState
		} else if ch == Sb {
			fsm.tc.cmdBuffer.WriteByte(ch)
			nextState = subnegState
		} else { // anything else
			fsm.tc.cmdBuffer.WriteByte(ch)
			fsm.tc.cmdHandlerWrapper(fsm.tc.handlerWriter, &fsm.tc.cmdBuffer)
			fsm.tc.cmdBuffer.Reset()
			nextState = dataState
		}
	case optionNegotiationState:
		fsm.tc.cmdBuffer.WriteByte(ch)
		opt := ch
		cmd := fsm.tc.cmdBuffer.Bytes()[0]
		fsm.tc.optionCallback(cmd, opt)
		fsm.tc.cmdBuffer.Reset()
		nextState = dataState
	case subnegState:
		if ch == Iac {
			nextState = subnegEndState
		} else {
			nextState = subnegState
			fsm.tc.cmdBuffer.WriteByte(ch)
		}
	case subnegEndState:
		if ch == Se {
			fsm.tc.cmdBuffer.WriteByte(ch)
			fsm.tc.cmdHandlerWrapper(fsm.tc.handlerWriter, &fsm.tc.cmdBuffer)
			fsm.tc.cmdBuffer.Reset()
			nextState = dataState
		} else if ch == Iac { // escaping IAC
			nextState = subnegState
			fsm.tc.cmdBuffer.WriteByte(ch)
		} else {
			nextState = errorState
		}
	case errorState:
		nextState = dataState
		log.Infof("Finite state machine is in an error state. This should not happen for correct telnet protocol syntax")
	}
	return nextState
}
