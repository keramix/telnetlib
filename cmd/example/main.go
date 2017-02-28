package main

import (
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

func handleKnownSuboptions(w io.Writer, b []byte) {
	log.Printf("Handling KNOWN SUBOPTIONS")
	var resp []byte
	suboptions := b[3 : len(b)-2]
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, KNOWN_SUBOPTIONS_2}...)
	resp = append(resp, suboptions...)
	resp = append(resp, telnetlib.IAC, telnetlib.SE)
	w.Write(resp)
}

func handleDoProxy(w io.Writer, b []byte) {
	log.Printf("Handling DO PROXY")
	var resp []byte
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, WILL_PROXY, telnetlib.IAC, telnetlib.SE}...)
	w.Write(resp)
}

func handleVmotionBegin(w io.Writer, b []byte) {
	log.Printf("Handling VMOTION BEGIN")
	seq := b[4 : len(b)-2]
	secret := make([]byte, 4)
	rand.Read(secret)
	var resp []byte
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, VMOTION_GOAHEAD}...)
	resp = append(resp, seq...)
	resp = append(resp, secret...)
	resp = append(resp, telnetlib.IAC, telnetlib.SE)
	w.Write(resp)
}

func handleVmotionPeer(w io.Writer, b []byte) {
	// this should send back the sequence only but I will try to send the sequence and the secret
	log.Printf("Handling VMOTION PEER")
	cookie := b[4 : len(b)-2]
	var resp []byte
	resp = append(resp, []byte{telnetlib.IAC, telnetlib.SB, VMWARE_EXT, VMOTION_PEER_OK}...)
	resp = append(resp, cookie...)
	resp = append(resp, telnetlib.IAC, telnetlib.SE)
	w.Write(resp)
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
		}
	}

	opts := telnetlib.TelnetOpts{
		Addr:        ":6779",
		ServerOpts:  []byte{telnetlib.BINARY, telnetlib.SGA, telnetlib.ECHO},
		ClientOpts:  []byte{telnetlib.BINARY, telnetlib.SGA, VMWARE_EXT},
		DataHandler: dhandler,
	}

	telnetlib.NewTelnetServer(opts).Serve()
}
