package main

import (
	"io"
	"log"

	"github.com/kreamyx/telnetlib"
)

var dhandler telnetlib.DataHandlerFunc
var chandler telnetlib.CmdHandlerFunc

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
				_, err := w.Write(buf)
				if err != nil {
					log.Printf("write error: %v", err)
				}
			}
		}
	}

	chandler = func(w io.Writer, r io.Reader) {

	}

	opts := telnetlib.TelnetOpts{
		Addr: ":6779",
		//ServerOpts:  []byte{0, 3, 1},
		//ClientOpts:  []byte{0, 3, 1},
		DataHandler: dhandler,
	}

	telnetlib.NewTelnetServer(opts).Serve()
}
