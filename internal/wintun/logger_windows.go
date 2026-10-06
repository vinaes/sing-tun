package wintun

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

type loggerLevel int

const (
	logInfo loggerLevel = iota
	logWarn
	logErr
)

// Wintun reports why it failed only through its logger: the error it returns
// is a bare Win32 code ("the system cannot find the file" when the adapter
// device never started). Warnings and errors go to stderr, which the host
// captures; info lines (driver version, adapter created) are dropped.
func writeLog(level loggerLevel, msg *uint16) {
	if level < logWarn {
		return
	}
	tag := "warn"
	if level >= logErr {
		tag = "error"
	}
	fmt.Fprintf(os.Stderr, "%s wintun %s: %s\n", time.Now().UTC().Format(time.RFC3339), tag, windows.UTF16PtrToString(msg))
}

func setupLogger(dll *lazyDLL) {
	var callback uintptr
	switch runtime.GOARCH {
	case "386":
		callback = windows.NewCallback(func(level loggerLevel, timestampLow, timestampHigh uint32, msg *uint16) int {
			writeLog(level, msg)
			return 0
		})
	case "arm":
		callback = windows.NewCallback(func(level loggerLevel, _, timestampLow, timestampHigh uint32, msg *uint16) int {
			writeLog(level, msg)
			return 0
		})
	case "amd64", "arm64":
		callback = windows.NewCallback(func(level loggerLevel, timestamp uint64, msg *uint16) int {
			writeLog(level, msg)
			return 0
		})
	default:
		return
	}
	proc := dll.NewProc("WintunSetLogger")
	if proc.Find() != nil {
		return
	}
	syscall.SyscallN(proc.Addr(), callback)
}
