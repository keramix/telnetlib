package telnetlib

const (
	/* Telnet Protocol Characters */
	Iac  byte = 255 // "Interpret As Command"
	Dont byte = 254
	Do   byte = 253
	Wont byte = 252
	Will byte = 251
	Null byte = 0

	Se  byte = 240 // Subnegotiation End
	Nop byte = 241 // No Operation
	Dm  byte = 242 // Data Mark
	Brk byte = 243 // Break
	Ip  byte = 244 // Interrupt process
	Ao  byte = 245 // Abort output
	Ayt byte = 246 // Are You There
	Ec  byte = 247 // Erase Character
	El  byte = 248 // Erase Line
	Ga  byte = 249 // Go Ahead
	Sb  byte = 250 // Subnegotiation Begin

	/* Telnet Options */
	Binary          byte = 0   // 8-bit data path
	Echo            byte = 1   // echo
	Rcp             byte = 2   // prepare to reconnect
	Sga             byte = 3   // suppress go ahead
	Nams            byte = 4   // approximate message size
	Status          byte = 5   // give status
	Tm              byte = 6   // timing mark
	Rcte            byte = 7   // remote controlled transmission and echo
	Naol            byte = 8   // negotiate about output line width
	Naop            byte = 9   // negotiate about output page size
	Naocrd          byte = 10  // negotiate about CR disposition
	Naohts          byte = 11  // negotiate about horizontal tabstops
	Naohtd          byte = 12  // negotiate about horizontal tab disposition
	Naoffd          byte = 13  // negotiate about formfeed disposition
	Naovts          byte = 14  // negotiate about vertical tab stops
	Naovtd          byte = 15  // negotiate about vertical tab disposition
	Naolfd          byte = 16  // negotiate about output LF disposition
	Xascii          byte = 17  // extended ascii character set
	Logout          byte = 18  // force logout
	Bm              byte = 19  // byte macro
	Det             byte = 20  // data entry terminal
	Supdup          byte = 21  // supdup protocol
	SupdupOutput    byte = 22  // supdup output
	SndLoc          byte = 23  // send location
	Ttype           byte = 24  // terminal type
	Eor             byte = 25  // end or record
	TuId            byte = 26  // TACACS user identification
	OutMrk          byte = 27  // output marking
	TtyLoc          byte = 28  // terminal location number
	Vt3270Regime    byte = 29  // 3270 regime
	X3Pad           byte = 30  // X.3 PAD
	Naws            byte = 31  // window size
	Tspeed          byte = 32  // terminal speed
	Lflow           byte = 33  // remote flow control
	LineMode        byte = 34  // Linemode option
	XDispLoc        byte = 35  // X Display Location
	Old_Environ     byte = 36  // Old - Environment variables
	Authentication  byte = 37  // Authenticate
	Encrypt         byte = 38  // Encryption option
	New_Environ     byte = 39  // New - Environment variables
	TN3270E         byte = 40  // TN3270E
	XAuth           byte = 41  // XAUTH
	Charset         byte = 42  // CHARSET
	Rsp             byte = 43  // Telnet Remote Serial Port
	ComPortOption   byte = 44  // Com Port Control Option
	SupLocalEcho    byte = 45  // Telnet Suppress Local Echo
	Tls             byte = 46  // Telnet Start TLS
	Kermit          byte = 47  // KERMIT
	SendUrl         byte = 48  // SEND-URL
	ForwardX        byte = 49  // FORWARD_X
	PragmaLogon     byte = 138 // TELOPT PRAGMA LOGON
	SspiLogin       byte = 139 // TELOPT SSPI LOGON
	PragmaHeartbeat byte = 140 // TELOPT PRAGMA HEARTBEAT
	ExtOptList      byte = 255 // Extended-Options-List
	NoOp            byte = 0
)
