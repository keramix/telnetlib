package main

import (
	"bytes"
	"crypto/rand"
	"io"
	"log"

	"github.com/kreamyx/telnetlib"
)

var dhandler telnetlib.DataHandlerFunc
var chandler telnetlib.CmdHandlerFunc

const (
	VMWARE_EXT               byte = 232
	KNOWN_SUBOPTIONS_1       byte = 0
	KNOWN_SUBOPTIONS_2       byte = 1
	UNKNOWN_SUBOPTION_RCVD_1 byte = 2
	UNKNOWN_SUBOPTION_RCVD_2 byte = 3
	VMOTION_BEGIN            byte = 40
	VMOTION_GOAHEAD          byte = 41
	VMOTION_NOTNOW           byte = 43
	VMOTION_PEER             byte = 44
	VMOTION_PEER_OK          byte = 45
	VMOTION_COMPLETE         byte = 46
	VMOTION_ABORT            byte = 48
	DO_PROXY                 byte = 70
	WILL_PROXY               byte = 71
	WONT_PROXY               byte = 73
	VM_VC_UUID               byte = 80
	GET_VM_VC_UUID           byte = 81
	VM_NAME                  byte = 82
	GET_VM_NAME              byte = 83
	VM_BIOS_UUID             byte = 84
	GET_VM_BIOS_UUID         byte = 85
	VM_LOCATION_UUID         byte = 86
	GET_VM_LOCATION_UUID     byte = 87
)

func isKnownSuboptions(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == KNOWN_SUBOPTIONS_1
}

func isDoProxy(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == DO_PROXY
}

func isVmotionBegin(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == VMOTION_BEGIN
}

func isVmotionPeer(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == VMOTION_PEER
}

func isVmotionComplete(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == VMOTION_COMPLETE
}

func isVmotionAbort(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == VMOTION_ABORT
}

func isVMName(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == VM_NAME
}

func isVMUUID(cmd []byte) bool {
	if len(cmd) < 3 {
		return false
	}
	return cmd[0] == telnetlib.SB && cmd[1] == VMWARE_EXT && cmd[2] == VM_VC_UUID
}

func getVMName() []byte {
	return []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, GET_VM_NAME, telnetlib.IAC, telnetlib.SE}
}

func getVMUUID() []byte {
	return []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, GET_VM_VC_UUID, telnetlib.IAC, telnetlib.SE}
}

func handleVMName(w io.Writer, b []byte) {
	log.Printf("Got VM Name: %v", string(b[3:len(b)-1]))
}

func handleVMUUID(w io.Writer, b []byte) {
	log.Printf("Got VM UUID: %v", string(b[3:len(b)-1]))
}

func handleKnownSuboptions(w io.Writer, b []byte) {
	log.Printf("Handling KNOWN SUBOPTIONS")
	var resp []byte
	suboptions := b[3 : len(b)-1]
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, KNOWN_SUBOPTIONS_2}...)
	resp = append(resp, suboptions...)
	resp = append(resp, telnetlib.IAC, telnetlib.SE)
	log.Printf("response: %v", resp)
	if bytes.IndexByte(suboptions, GET_VM_VC_UUID) != -1 && bytes.IndexByte(suboptions, VM_VC_UUID) != -1 {
		resp = append(resp, getVMUUID()...)
	}
	if bytes.IndexByte(suboptions, VM_NAME) != -1 && bytes.IndexByte(suboptions, GET_VM_NAME) != -1 && bytes.IndexByte(suboptions, GET_VM_VC_UUID) != -1 && bytes.IndexByte(suboptions, VM_VC_UUID) != -1 {
		resp = append(resp, getVMName()...)
	}
	w.Write(resp)
}

func handleDoProxy(w io.Writer, b []byte) {
	log.Printf("Handling DO PROXY")
	var resp []byte
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, WILL_PROXY, telnetlib.IAC, telnetlib.SE}...)
	log.Printf("response: %v", resp)
	w.Write(resp)
}

func handleVmotionBegin(w io.Writer, b []byte) {
	log.Printf("Handling VMOTION BEGIN")
	seq := b[3 : len(b)-1]
	secret := make([]byte, 4)
	rand.Read(secret)
	var resp []byte
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, VMOTION_GOAHEAD}...)
	resp = append(resp, seq...)
	resp = append(resp, secret...)
	resp = append(resp, telnetlib.IAC, telnetlib.SE)
	log.Printf("response: %v", resp)
	w.Write(resp)
}

func handleVmotionPeer(w io.Writer, b []byte) {
	// this should send back the sequence only but I will try to send the sequence and the secret
	log.Printf("Handling VMOTION PEER")
	cookie := b[3 : len(b)-1]
	var resp []byte
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, VMOTION_PEER_OK}...)
	resp = append(resp, cookie...)
	resp = append(resp, telnetlib.IAC, telnetlib.SE)
	log.Printf("response: %v", resp)
	w.Write(resp)
}

func handleVmotionComplete(w io.Writer, b []byte) {
	log.Printf("vMotion is Complete")
}

func handleVmotionAbort(w io.Writer, b []byte) {
	log.Printf("Aborting vMotion")
}

func main() {
	// This is an echoHandler
	dhandler = func(w io.Writer, r io.Reader) {
		log.Printf("************* STARTED DATA HANDLER ***************")
		for {
			buf := make([]byte, 512)
			n1, err := r.Read(buf)
			if err != nil {
				//log.Printf("error: %v", err)
			}
			if n1 != 0 {
				//_, err := w.Write(buf)
				if err != nil {
					log.Printf("write error: %v", err)
				}
			}
		}
	}

	chandler = func(w io.Writer, r io.Reader) {
		b := make([]byte, 512)
		log.Printf("command recieved")
		cmdLen, err := r.Read(b)
		if err != nil && err != io.EOF {
			panic(err)
		}
		b = b[:cmdLen]
		if isKnownSuboptions(b) {
			handleKnownSuboptions(w, b)
		} else if isDoProxy(b) {
			handleDoProxy(w, b)
		} else if isVmotionBegin(b) {
			handleVmotionBegin(w, b)
		} else if isVmotionPeer(b) {
			handleVmotionPeer(w, b)
		} else if isVMName(b) {
			handleVMName(w, b)
		} else if isVMUUID(b) {
			handleVMUUID(w, b)
		} else if isVmotionComplete(b) {
			handleVmotionComplete(w, b)
		} else if isVmotionAbort(b) {
			handleVmotionAbort(w, b)
		}
	}

	// FIXME: Handle the case when cmdHandler and DataHandler are nil
	// Default handler should just read bytes and do nothing with them
	opts := telnetlib.TelnetOpts{
		Addr:       ":6779",
		ServerOpts: []byte{telnetlib.BINARY, telnetlib.SGA, telnetlib.ECHO},
		ClientOpts: []byte{telnetlib.BINARY, telnetlib.SGA},
		//DataHandler: dhandler,
		//CmdHandler:  chandler,
	}
	srvr := telnetlib.NewTelnetServer(opts)
	for {
		_, err := srvr.Accept()
		if err != nil {
			panic(err)
		}
		//time.Sleep(10 * time.Second)
		//conn.Close()
	}
}
