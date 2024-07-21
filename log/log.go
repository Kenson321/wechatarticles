package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

var debugLog *log.Logger
var debug bool

func init() {
	debugLog = log.New(os.Stderr, "", log.LstdFlags)
	debug = false
}

func SetDebug(flag bool, file string) {
	lf, err := os.Create(file)
	if err != nil {
		fmt.Println("打开文件失败", err)
	} else {
		w := io.MultiWriter(os.Stderr, lf)
		debugLog = log.New(w, "", log.LstdFlags)
	}

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

func Info(v ...interface{}) {
	debugLog.Println(v)
}

func Error(v ...interface{}) {
	debugLog.Println("========== ========== ========== ==========")
	debugLog.Println(v)
	debugLog.Println("========== ========== ========== ==========")
}
