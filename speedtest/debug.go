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
//
// It writes to standard error, so debug output never mixes into results written to standard output, such as JSON.
//
// Returns:
//   - *Debug: the logger, disabled until [Debug.Enable] is called.
func NewDebug() *Debug {
	return &Debug{dbg: log.New(os.Stderr, "[DBG]", log.Ldate|log.Ltime)}
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
