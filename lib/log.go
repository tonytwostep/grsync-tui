package lib

import (
	"bytes"
	"github.com/rivo/tview"
)

var (
	log = bytes.Buffer{}
)

func WriteLog(message string, logBox *tview.TextView) {
	log.WriteString(message + "\n")
	logBox.ScrollToEnd()
}

func SetLogText(logBox *tview.TextView) {
	logBox.SetText(log.String())
}
