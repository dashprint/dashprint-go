package main

import (
	"log"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/jacobsa/go-serial/serial"
)

const (
	STATE_STOPPED      = iota
	STATE_DISCONNECTED = iota
	STATE_INITIALIZING = iota
	STATE_CONNECTED    = iota
)

const (
	kTIOCEXCL = 0x540C
	kNCCS     = 19
	kTCGETS2  = 0x802C542A
	kTCSETS2  = 0x402C542B
	kHUPCL    = 0x00004000
)

type PrinterSettings struct {
	Name       string    `json:"name"`
	DevicePath string    `json:"devicePath"`
	UniqueName string    `json:"uniqueName"`
	BaudRate   uint      `json:"baudRate"`
	Stopped    bool      `json:"stopped"`
	PrintArea  PrintArea `json:"printArea"`
}

type Printer struct {
	PrinterSettings
	state         int
	channel       chan int
	nextLineNo    int
	sendWaitGroup sync.WaitGroup
}

type PrintArea struct {
	Width, Height, Depth uint
}

func LoadPrinter(settings PrinterSettings) *Printer {
	p := &Printer{}
	p.PrinterSettings = settings
	return p
}

func (p *Printer) Start() {
	// TODO: add mutex
	if p.state != STATE_STOPPED {
		log.Printf("Printer %s is not stopped, but Start() was called\n", p.UniqueName)
		return
	}

	p.setState(STATE_DISCONNECTED)
	p.channel = make(chan int)
	go p.mainLoop()
}

func (p *Printer) setState(state int) {
	// TODO: notifications
	p.state = state
}

func (p *Printer) mainLoop() {
	var port *os.File

	p.sendWaitGroup = sync.WaitGroup{}
	p.sendWaitGroup.Add(1)

	for {
		if port != nil {
			port.Close()
		}

		port = p.doConnect()

		// If connection failed, wait before reconnecting
		if port == nil {
			select {
			case <-time.After(5 * time.Second):
				continue
			case <-p.channel:
				break
			}
		}

		setNoResetOnReopen(port)

		log.Printf("[%s] Successfully opened serial port\n", p.UniqueName)
		p.setState(STATE_INITIALIZING)

		time.Sleep(1000)

		// Read and drop anything found on the serial interface
		port.Read(make([]byte, 1000))

		// Send initial commands
		p.SendCommand("M110 N0", nil)
		p.nextLineNo = 1

		// Get printer information
		p.SendCommand("M115", func(reply []string) {

		})
	}

	if port != nil {
		port.Close()
	}
}

func (p *Printer) SendCommand(command string, callback func(reply []string)) {
	p.sendWaitGroup.Wait()
	defer p.sendWaitGroup.Done()

	// TODO: do sending
}

func (p *Printer) doConnect() *os.File {
	options := serial.OpenOptions{
		PortName: p.DevicePath,
		BaudRate: p.BaudRate,
		DataBits: 8,
		StopBits: 1,
	}

	port, err := serial.Open(options)
	if err != nil {
		return nil
	}

	var file *os.File = port.(*os.File)
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), kTIOCEXCL, 0)

	return file
}

type cc_t byte
type speed_t uint32
type tcflag_t uint32
type termios2 struct {
	c_iflag  tcflag_t    // input mode flags
	c_oflag  tcflag_t    // output mode flags
	c_cflag  tcflag_t    // control mode flags
	c_lflag  tcflag_t    // local mode flags
	c_line   cc_t        // line discipline
	c_cc     [kNCCS]cc_t // control characters
	c_ispeed speed_t     // input speed
	c_ospeed speed_t     // output speed
}

func setNoResetOnReopen(file *os.File) {
	var to termios2

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(kTCGETS2), uintptr(unsafe.Pointer(&to)))

	if errno == 0 && (to.c_cflag&kHUPCL) != 0 {
		to.c_cflag &^= kHUPCL
		syscall.Syscall(syscall.SYS_IOCTL, uintptr(file.Fd()), uintptr(kTCSETS2), uintptr(unsafe.Pointer(&to)))
	}
}
