package log 

import (
	"log"
	"os"
)

var debugLog *log.Logger
var debug bool

func init() {
	debugLog = log.New(os.Stderr, "", log.LstdFlags)
}

func Println(v ...interface{}) {
	debugLog.Println(v)
}

func SetDebug(flag bool) {
	debug = flag
	if debug {
		debugLog.SetFlags(debugLog.Flags() | log.Llongfile)
	}
}

func Debug(v ...interface{}) {
	if debug {
		debugLog.Println(v)
	}
}
