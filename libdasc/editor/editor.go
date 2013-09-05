// package editor implements a full-feature line editor.
package editor

import (
	"os"
	"fmt"
	"bufio"
	"unicode"
	"unicode/utf8"
	"./tty"
)

type Editor struct {
	savedTermios *tty.Termios
}

type LineRead struct {
	Line string
	Eof bool
	Err error
}

var savedTermios *tty.Termios

func Init() (*Editor, error) {
	var err error
	editor := &Editor{}
	editor.savedTermios, err = tty.NewTermiosFromFd(0)
	if err != nil {
		return nil, fmt.Errorf("Can't get terminal attribute of stdin: %s", err)
	}
	term := editor.savedTermios.Copy()

	term.SetIcanon(false)
	term.SetEcho(false)
	term.SetMin(1)
	term.SetTime(0)

	err = term.ApplyToFd(0)
	if err != nil {
		return nil, fmt.Errorf("Can't set up terminal attribute of stdin: %s", err)
	}

	fmt.Print("\033[?7l")
	return editor, nil
}

func (ed *Editor) Cleanup() error {
	fmt.Print("\033[?7h")

	err := ed.savedTermios.ApplyToFd(0)
	if err != nil {
		return fmt.Errorf("Can't restore terminal attribute of stdin: %s", err)
	}
	ed.savedTermios = nil
	return nil
}

func (ed *Editor) beep() {
}

func (ed *Editor) tip(s string) {
	fmt.Printf("\n%s\033[A", s)
}

func (ed *Editor) tipf(format string, a ...interface{}) {
	ed.tip(fmt.Sprintf(format, a...))
}

func (ed *Editor) clearTip() {
	fmt.Printf("\n\033[K\033[A")
}

func (ed *Editor) refresh(prompt, text string) (newlines int, err error) {
	w := newWriter()
	defer func() {
		newlines = w.line
	}()
	for _, r := range prompt {
		err = w.write(r)
		if err != nil {
			return
		}
	}
	var indent int
	if w.col * 2 < w.width {
		indent = w.col
	}
	for _, r := range text {
		err = w.write(r)
		if err != nil {
			return
		}
		if w.col == 0 {
			for i := 0; i < indent; i++ {
				err = w.write(' ')
				if err != nil {
					return
				}
			}
		}
	}
	return
}

func (ed *Editor) ReadLine(prompt string) (lr LineRead) {
	stdin := bufio.NewReaderSize(os.Stdin, 0)
	line := ""

	newlines := 0

	for {
		if newlines > 0 {
			fmt.Printf("\033[%dA", newlines)
		}
		fmt.Printf("\r\033[J")

		newlines, _ = ed.refresh(prompt, line)

		r, _, err := stdin.ReadRune()
		if err != nil {
			return LineRead{Err: err}
		}

		switch {
		case r == '\n':
			ed.clearTip()
			fmt.Println()
			return LineRead{Line: line}
		case r == 0x7f: // Backspace
			if l := len(line); l > 0 {
				_, w := utf8.DecodeLastRuneInString(line)
				line = line[:l-w]
			} else {
				ed.beep()
			}
		case r == 0x15: // ^U
			line = ""
		case r == 0x4 && len(line) == 0: // ^D
			return LineRead{Eof: true}
		case r == 0x2: // ^B
			fmt.Printf("\033[D")
		case r == 0x6: // ^F
			fmt.Printf("\033[C")
		case unicode.IsGraphic(r):
			line += string(r)
		default:
			ed.tipf("Non-graphic: %#x", r)
		}
	}

	panic("unreachable")
}
