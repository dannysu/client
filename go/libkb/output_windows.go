// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

// +build windows

package libkb

import (
	"bytes"
	"io"
	"os"
	"sync"
	"syscall"
)

var (
	kernel32DLL                 = syscall.NewLazyDLL("kernel32.dll")
	setConsoleTextAttributeProc = kernel32DLL.NewProc("SetConsoleTextAttribute")
	stdOutMutex                 sync.Mutex
	stdErrMutex                 sync.Mutex
)

type WORD uint16

const (
	// Character attributes
	// Note:
	// -- The attributes are combined to produce various colors (e.g., Blue + Green will create Cyan).
	//    Clearing all foreground or background colors results in black; setting all creates white.
	// See https://msdn.microsoft.com/en-us/library/windows/desktop/ms682088(v=vs.85).aspx#_win32_character_attributes.
	fgBlack     WORD = 0x0000
	fgBlue      WORD = 0x0001
	fgGreen     WORD = 0x0002
	fgCyan      WORD = 0x0003
	fgRed       WORD = 0x0004
	fgMagenta   WORD = 0x0005
	fgYellow    WORD = 0x0006
	fgWhite     WORD = 0x0007
	fgIntensity WORD = 0x0008
	fgMask      WORD = 0x000F

	bgBlack     WORD = 0x0000
	bgBlue      WORD = 0x0010
	bgGreen     WORD = 0x0020
	bgCyan      WORD = 0x0030
	bgRed       WORD = 0x0040
	bgMagenta   WORD = 0x0050
	bgYellow    WORD = 0x0060
	bgWhite     WORD = 0x0070
	bgIntensity WORD = 0x0080
	bgMask      WORD = 0x00F0
)

var codesWin = map[byte]WORD{
	0:  fgWhite | bgBlack,     //	"reset":
	1:  fgIntensity | fgWhite, //CpBold          = CodePair{1, 22}
	22: fgWhite,               //	UnBold:        // Just assume this means reset to white fg
	39: fgWhite,               //	"resetfg":
	49: fgWhite,               //	"resetbg":        // Just assume this means reset to white fg
	30: fgBlack,               //	"black":
	31: fgRed,                 //	"red":
	32: fgGreen,               //	"green":
	33: fgYellow,              //	"yellow":
	34: fgBlue,                //	"blue":
	35: fgMagenta,             //	"magenta":
	36: fgCyan,                //	"cyan":
	37: fgWhite,               //	"white":
	90: fgWhite,               //	"grey":
	40: bgBlack,               //	"bgBlack":
	41: bgRed,                 //	"bgRed":
	42: bgGreen,               //	"bgGreen":
	43: bgYellow,              //	"bgYellow":
	44: bgBlue,                //	"bgBlue":
	45: bgMagenta,             //	"bgMagenta":
	46: bgCyan,                //	"bgCyan":
	47: bgWhite,               //	"bgWhite":
}

// Return our writer so we can override Write()
func OutputWriter() io.Writer {
	return &ColorWriter{os.Stdout, os.Stdout.Fd(), &stdOutMutex}
}

// Return our writer so we can override Write()
func ErrorWriter() io.Writer {
	return &ColorWriter{os.Stderr, os.Stderr.Fd(), &stdErrMutex}
}

type ColorWriter struct {
	w     io.Writer
	fd    uintptr
	mutex *sync.Mutex
}

// Fd returns the underlying file descriptor. This is mainly to
// support checking whether it's a terminal or not.
func (cw *ColorWriter) Fd() uintptr {
	return cw.fd
}

// Rough emulation of Ansi terminal codes.
// Parse the string, pick out the codes, and convert to
// calls to SetConsoleTextAttribute.
//
// This function must mix calls to SetConsoleTextAttribute
// with separate Write() calls, so to prevent pieces of colored
// strings from being interleaved by unsynchronized goroutines,
// we unfortunately must lock a mutex.
//
// Notice this is only necessary for what is now called
// "legacy" terminal mode:
// https://technet.microsoft.com/en-us/library/mt427362.aspx?f=255&MSPPError=-2147217396
func (cw *ColorWriter) Write(p []byte) (n int, err error) {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	var totalWritten = len(p)
	ctlStart := []byte{0x1b, '['}

	for nextIndex := 0; len(p) > 0; {

		var controlIndex int = -1
		// search for the next control code
		nextIndex = bytes.Index(p, ctlStart)
		if nextIndex == -1 {
			nextIndex = len(p)
		} else {
			controlIndex = nextIndex + 2
		}

		cw.w.Write(p[0:nextIndex])

		if controlIndex != -1 {
			// The control code is written as separate ascii digits. usually 2,
			// ending with 'm'.
			// Stop at 4 as a sanity check.
			controlCode := p[controlIndex] - '0'
			controlIndex++
			for i := 0; p[controlIndex] != 'm' && i < 4; i++ {
				controlCode *= 10
				controlCode += p[controlIndex] - '0'
				controlIndex++
			}
			if code, ok := codesWin[controlCode]; ok {
				setConsoleTextAttribute(cw.fd, code)
			}
			nextIndex = controlIndex + 1
		}
		p = p[nextIndex:]
	}
	return totalWritten, nil
}

// setConsoleTextAttribute sets the attributes of characters written to the
// console screen buffer by the WriteFile or WriteConsole function.
// See http://msdn.microsoft.com/en-us/library/windows/desktop/ms686047(v=vs.85).aspx.
func setConsoleTextAttribute(handle uintptr, attribute WORD) error {
	r1, r2, err := setConsoleTextAttributeProc.Call(handle, uintptr(attribute), 0)
	use(attribute)
	return checkError(r1, r2, err)
}

// checkError evaluates the results of a Windows API call and returns the error if it failed.
func checkError(r1, r2 uintptr, err error) error {
	// Windows APIs return non-zero to indicate success
	if r1 != 0 {
		return nil
	}

	// Return the error if provided, otherwise default to EINVAL
	if err != nil {
		return err
	}
	return syscall.EINVAL
}

// use is a no-op, but the compiler cannot see that it is.
// Calling use(p) ensures that p is kept live until that point.
func use(p interface{}) {}
