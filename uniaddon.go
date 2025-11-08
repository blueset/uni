package main

import (
	"C"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"zgo.at/zli"
)

// DO NOT REMOVE THIS COMMENT. Needed for gonacli
//export jsexec
func jsexec(_argsJSON *C.char) *C.char {
	// argsJSON string type，return string type
	argsJSON := C.GoString(_argsJSON)

	// Parse JSON array of strings
	var args []string
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return C.CString(fmt.Sprintf("Error parsing JSON: %v", err))
	}

	// Capture stdout
	oldStdout := zli.Stdout
	oldStderr := zli.Stderr
	oldWantColor := zli.WantColor
	oldIsTerminal := zli.IsTerminal
	oldIsTerm := isTerm
	buf := new(strings.Builder)
	zli.Stdout = buf
	zli.Stderr = buf
	zli.WantColor = false
	zli.IsTerminal = func(f uintptr) bool { return false }
	isTerm = false
	defer func() {
		zli.Stdout = oldStdout
		zli.Stderr = oldStderr
		zli.WantColor = oldWantColor
		zli.IsTerminal = oldIsTerminal
		isTerm = oldIsTerm
	}()

	// Parse flags using the provided args
	flag := zli.NewFlags(args)
	var (
		compact  = flag.Bool(false, "c", "compact", "q", "quiet")
		help     = flag.Bool(false, "h", "help")
		versionF = flag.Bool(false, "v", "version")
		rawF     = flag.Bool(false, "r", "raw")
		or       = flag.Bool(false, "o", "or")
		limitF   = flag.Int(0, "l", "limit")
		formatF  = flag.String(defaultFormat, "format", "f")
		tone     = flag.String("", "t", "tone", "tones")
		gender   = flag.String("person", "g", "gender", "genders")
		asF      = flag.String("list", "a", "as")
		jsonF    = flag.Bool(false, "json", "j")
	)
	
	if err := flag.Parse(); err != nil {
		return C.CString(fmt.Sprintf("Error parsing flags: %v", err))
	}

	if versionF.Set() {
		fmt.Fprint(buf, version)
		return C.CString(buf.String())
	}
	
	if help.Set() {
		fmt.Fprint(buf, usage)
		return C.CString(buf.String())
	}

	cmd, err := flag.ShiftCommand("list", "ls", "identify", "print", "search", "emoji", "help", "version")
	if cmd == "ls" {
		cmd = "list"
	}
	
	switch cmd {
	case "":
		fmt.Fprint(buf, usageShort)
		return C.CString(buf.String())
	case "help":
		fmt.Fprint(buf, usage)
		return C.CString(buf.String())
	case "version":
		fmt.Fprint(buf, version)
		return C.CString(buf.String())
	}

	var (
		as    = parseAsFlags(compact, asF, jsonF)
		quiet = compact.Set()
		raw   = rawF.Set()
		cmdArgs  = flag.Args
	)
	
	if cmd == "print" {
		cmdArgs, err = zli.InputOrArgs(cmdArgs, " \t\n", quiet)
		if err != nil {
			return C.CString(fmt.Sprintf("Error: %v", err))
		}
	} else if cmd != "list" {
		cmdArgs, err = zli.InputOrArgs(cmdArgs, "", quiet)
		if err != nil {
			return C.CString(fmt.Sprintf("Error: %v", err))
		}
	}

	format := formatF.String()
	if !formatF.Set() && cmd == "emoji" {
		format = defaultEmojiFormat
	}

	if formatF.String() == "all" {
		format = allFormat
		if cmd == "emoji" {
			format = allEmojiFormat
		}
	}
	if strings.HasPrefix(formatF.String(), "+") {
		format = defaultCompact
		if cmd == "emoji" {
			format = defaultEmojiCompact
		}
		format += " " + formatF.String()[1:]
	}
	// Replace %name shortcut with %(name l:auto)
	format = regexp.MustCompile(`%[a-z0-9-]+`).ReplaceAllStringFunc(format, func(s string) string {
		return "%(" + s[1:] + " l:auto)"
	})

	switch cmd {
	case "list":
		err = list(cmdArgs, as)
	case "identify":
		err = identify(cmdArgs, format, raw, as)
	case "search":
		err = search(cmdArgs, format, raw, as, or.Bool(), limitF.Int())
	case "print":
		err = print(cmdArgs, format, raw, as)
	case "emoji":
		err = emoji(cmdArgs, format, raw, as, or.Bool(), parseToneFlag(tone.String()), parseGenderFlag(gender.String()))
	}
	
	if err != nil {
		if !(err == errNoMatches && quiet) {
			return C.CString(fmt.Sprintf("Error: %v", err))
		}
	}

	return C.CString(buf.String())
}
