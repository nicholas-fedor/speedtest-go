package speedtest

import (
	"log"
	"os"
)

// Debug is a simple debug logging utility.
type Debug struct {
	dbg  *log.Logger
	flag bool
}

// NewDebug creates a new debug logger.
func NewDebug() *Debug {
	return &Debug{dbg: log.New(os.Stdout, "[DBG]", log.Ldate|log.Ltime)}
}

// Enable enables debug logging.
func (d *Debug) Enable() {
	d.flag = true
}

// Println prints debug messages if enabled.
func (d *Debug) Println(v ...any) {
	if d.flag {
		d.dbg.Println(v...)
	}
}

// Printf prints formatted debug messages if enabled.
func (d *Debug) Printf(format string, v ...any) {
	if d.flag {
		d.dbg.Printf(format, v...)
	}
}

var dbg = NewDebug()
