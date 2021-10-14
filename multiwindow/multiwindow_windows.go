package multiwindow

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

func New(count int) ([]*Window, error) {
	hnd, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return nil, err
	}

	{
		var mode uint32
		err = windows.GetConsoleMode(hnd, &mode)
		if err != nil {
			return nil, err
		}

		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING

		err = windows.SetConsoleMode(hnd, mode)
		if err != nil {
			return nil, err
		}
	}

	var info windows.ConsoleScreenBufferInfo
	err = windows.GetConsoleScreenBufferInfo(hnd, &info)
	if err != nil {
		return nil, err
	}

	messages := make(chan windowMessage, 10)
	ret := make([]*Window, count)
	for i := range ret {
		ret[i] = &Window{
			id:       i,
			messages: messages,
			done:     make(chan struct{}),
		}
	}

	go func() {
		fmt.Print("\x1b[?1049h") // Alternative Buffer
		sectionHeight := int(info.Window.Bottom-info.Window.Top+1)/count - 3
		sectionWidth := int(info.Window.Right-info.Window.Left) + 1
		bufs := make([]*cursorBuffer, count)
		for i := range bufs {
			bufs[i] = newCursorBuffer(sectionWidth, sectionHeight)
		}
		titles := make([]string, count)
		errorStates := make([]bool, count)

		signals := make(chan os.Signal)
		signal.Notify(signals, os.Interrupt, os.Kill)
		t := time.Tick(time.Second / 30)

		update := false
		for {
			select {
			case <-signals:
				fmt.Print("\x1b[?1049l") // Return to primary buffer
				fmt.Println("\x1b[0m")   // Reset text format
				os.Exit(0)
			case msg := <-messages:
				switch {
				case msg.clear:
					bufs[msg.id].Clear()
					errorStates[msg.id] = false
				case msg.errorState:
					errorStates[msg.id] = true
				case msg.title != "":
					titles[msg.id] = msg.title
				case msg.b != nil:
					bufs[msg.id].Write(msg.b)
					ret[msg.id].done <- struct{}{}
				}
				update = true
			case <-t:
				if update {
					update = false

					fmt.Print("\x1b[0;0H") // move to 0,0
					for i, cb := range bufs {
						if errorStates[i] {
							fmt.Print("\x1b[41m\x1b[30m")
						} else {
							fmt.Print("\x1b[107m\x1b[30m")
						}
						for j := 0; j < sectionWidth; j++ {
							fmt.Print(" ")
						}
						for j := 0; j < 3; j++ {
							fmt.Print(" ")
						}
						fmt.Print(titles[i])
						for j := len(titles[i]) + 3; j < sectionWidth; j++ {
							fmt.Print(" ")
						}
						for j := 0; j < sectionWidth; j++ {
							fmt.Print(" ")
						}
						fmt.Print("\x1b[0m")
						cb.WriteToScreen()
					}
				}
			}
		}
	}()

	return ret, nil
}

type windowMessage struct {
	id         int
	clear      bool
	errorState bool
	title      string
	b          []byte
}

type Window struct {
	id       int
	messages chan windowMessage
	bufs     *cursorBuffer
	done     chan struct{}
}

func (s *Window) Write(b []byte) (int, error) {
	s.messages <- windowMessage{
		id: s.id,
		b:  b,
	}
	<-s.done
	return len(b), nil
}

func (s *Window) Section(str string) {
	fmt.Fprintln(s, "===", str)
}

func (s *Window) Clear() {
	s.messages <- windowMessage{
		id:    s.id,
		clear: true,
	}
}

func (s *Window) RunWithOutput(cmd *exec.Cmd) error {
	cmd.Stdout = s
	cmd.Stderr = s
	return cmd.Run()
}

func (s *Window) Title(title string) {
	s.messages <- windowMessage{
		id:    s.id,
		title: title,
	}
}

func (s *Window) ErrorState() {
	s.messages <- windowMessage{
		id:         s.id,
		errorState: true,
	}
}

type cursorBuffer struct {
	width  int
	height int
	x      int
	y      int
	bufs   [][]byte
}

func newCursorBuffer(width, height int) *cursorBuffer {
	cb := &cursorBuffer{
		width:  width,
		height: height,
		bufs:   make([][]byte, height),
	}

	for i := range cb.bufs {
		cb.bufs[i] = make([]byte, width)
	}
	cb.Clear()

	return cb
}

func (cb *cursorBuffer) Write(b []byte) (int, error) {
	for _, bit := range b {
		switch {
		case bit == '\r':
			// ignore
		case bit == '\n':
			for ; cb.x < cb.width; cb.x++ {
				cb.bufs[cb.y][cb.x] = ' '
			}
		case bit >= ' ' && bit <= '~': // all printable ascii characters
			cb.bufs[cb.y][cb.x] = bit
			cb.x++
		default:
			cb.bufs[cb.y][cb.x] = '?'
			cb.x++

		}
		if cb.x >= cb.width {
			cb.x = 0
			cb.y = (cb.y + 1) % cb.height
		}
	}
	if cb.x > 0 {
		for i := cb.x; i < cb.width; i++ {
			cb.bufs[cb.y][i] = ' '
		}
	}

	return len(b), nil
}

func (cb *cursorBuffer) WriteToScreen() {
	sy := cb.y
	y := sy
	for {
		os.Stdout.Write(cb.bufs[y])

		y++
		y %= cb.height
		if y == sy {
			break
		}
	}
}

func (cb *cursorBuffer) Clear() {
	for i := range cb.bufs {
		for j := range cb.bufs[i] {
			cb.bufs[i][j] = ' '
		}
	}
}
