package main

import (
	"io"
	"log"

	"github.com/kreamyx/telnetlib"
)

var dhandler telnetlib.DataHandlerFunc
var chandler telnetlib.CmdHandlerFunc

const VMWARE_EXT byte = 232

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
		_, err := r.Read(b)
		if err != nil && err != io.EOF {
			panic(err)
		}
		log.Printf("command: %v", b)
	}

	opts := telnetlib.TelnetOpts{
		Addr:        ":6779",
		ServerOpts:  []byte{telnetlib.BINARY, telnetlib.SGA, telnetlib.ECHO},
		ClientOpts:  []byte{telnetlib.BINARY, telnetlib.SGA, VMWARE_EXT},
		DataHandler: dhandler,
	}

	telnetlib.NewTelnetServer(opts).Serve()
}
