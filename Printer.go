package main

import (
	"log"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
	"bufio"
	"errors"
	"strings"
	"fmt"
	"strconv"
	"unicode"

	"github.com/jacobsa/go-serial/serial"
)

const (
	STATE_STOPPED      = iota
	STATE_DISCONNECTED = iota
	STATE_INITIALIZING = iota
	STATE_CONNECTED    = iota
)

const (
	MAX_LINENO = 10000
	DATA_TIMEOUT = 5000 // 5 seconds
	RECONNECT_TIMEOUT = 1000 // 1 second
	MAX_TEMPERATURE_HISTORY = 30 // 30 mintues
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
	
	// Channel for stopping the printer
	channel       chan int
	nextLineNo    int
	sendWaitGroup sync.WaitGroup
	
	// Lock for start/stop ops
	lock          sync.Mutex
	
	listenersLock sync.RWMutex
	listeners     map[PrinterListener]bool
	executingCommand string
	
	// Channel for the reading goroutine
	readChannel   chan *string
	
	port          *os.File
	baseParameters map[string]string
}

type PrinterListener interface {
	onPrinterStateChanged(oldState int, newState int)
}

type PrintArea struct {
	Width, Height, Depth uint
}

func LoadPrinter(settings PrinterSettings) *Printer {
	p := &Printer{}
	p.PrinterSettings = settings
	p.listeners = make(map[PrinterListener]bool)
	return p
}

func (p *Printer) Start() {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	if p.state != STATE_STOPPED {
		log.Printf("Printer %s is not stopped, but Start() was called\n", p.UniqueName)
		return
	}

	p.setState(STATE_DISCONNECTED)
	p.channel = make(chan int, 1)
	p.readChannel = make(chan *string)
	
	go p.mainLoop()
}

func (p *Printer) Stop() {
	p.lock.Lock()
	defer p.lock.Unlock()
	
	if p.state != STATE_STOPPED {
		p.channel <- 0
		p.state = STATE_STOPPED
	}
}

func stateString(state int) string {
	switch state {
		case STATE_STOPPED:
			return "stopped"
		case STATE_DISCONNECTED:
			return "disconnected"
		case STATE_INITIALIZING:
			return "initializing"
		case STATE_CONNECTED:
			return "connected"
		default:
			return "???"
	}
}

func (p *Printer) setState(state int) {
	oldState := p.state
	p.state = state
	
	log.Printf("[%s] State %s -> %s\n", stateString(oldState), stateString(state))
	
	listeners := p.getListeners()
	for cb, _ := range listeners {
		go cb.onPrinterStateChanged(oldState, state)
	}
}

// Get a copy of registered listeners
func (p *Printer) getListeners() (rv map[PrinterListener]bool) {
	
	p.listenersLock.RLock()
	defer p.listenersLock.RUnlock()
	
	rv = make(map[PrinterListener]bool)
	for k, v := range p.listeners {
		rv[k] = v
	}
	
	return
}

func (p *Printer) AddListener(l PrinterListener) {
	p.listenersLock.Lock()
	defer p.listenersLock.Unlock()
	
	p.listeners[l] = true
}

func (p *Printer) RemoveListener(l PrinterListener) {
	p.listenersLock.Lock()
	defer p.listenersLock.Unlock()
	
	delete(p.listeners, l)
}

func (p *Printer) waitBeforeReconnect() bool {
	select {
		case <-time.After(RECONNECT_TIMEOUT):
			return true
		case <-p.channel:
			return false
	}
}

func (p *Printer) mainLoop() {

	p.sendWaitGroup = sync.WaitGroup{}
	p.sendWaitGroup.Add(1)

	for {
		if p.port != nil {
			p.port.Close()
		}

		p.port = p.doConnect()

		// If connection failed, wait before reconnecting
		if p.port == nil {
			if !p.waitBeforeReconnect() {
				return
			} else {
				continue
			}
		}

		setNoResetOnReopen(p.port)

		log.Printf("[%s] Successfully opened serial port\n", p.UniqueName)
		p.setState(STATE_INITIALIZING)

		time.Sleep(1000)

		// Read and drop anything found on the serial interface
		p.port.Read(make([]byte, 1000))
		
		go p.readRoutine(p.port)

		// Send initial commands
		p.SendCommand("M110 N0", nil)
		p.nextLineNo = 1

		// Get printer information
		p.SendCommand("M115", func(reply []string, err error) {
			if err != nil {
				p.scheduleReconnection()
				return
			} else if len(reply) >= 2 {
				p.baseParameters = kvParse(reply[len(reply) - 2])
				log.Printf("[%s] Base printer params: %v\n", p.UniqueName, p.baseParameters)
			}
			
			p.setState(STATE_CONNECTED)
			// TODO: query temperature
		})
		break
	}
}

// Correctly parse key:value pairs returned by 3D printers
func kvParse(line string) map[string]string {
	kv := make(map[string]string)
	pos := 0
	
	keyPositions := make([][2]int, 0)
	
	for {
		pos = strings.Index(line[pos:], ":")
		if pos == -1 {
			break
		}
		
		x := pos-1
		
		for x >= 0 && !unicode.IsSpace(rune(line[x])) {
			if line[x] == ':' {
				goto Ignore;
			}
			
			x--
		}
		
		keyPositions = append(keyPositions, [2]int{ x+1, pos })
Ignore:
		pos++
	}
	
	for index, keyPos := range keyPositions {
		var value string
		key := line[keyPos[0]:keyPos[1]]
		
		if index + 1 < len(keyPositions) {
			value = line[keyPos[1]+1 : keyPositions[index+1][0]-1]
		} else {
			value = line[keyPos[1]+1:]
		}
		
		kv[key] = value
	}
	
	return kv
}

func (p *Printer) readRoutine(port *os.File) {
	var reader *bufio.Reader = bufio.NewReader(port)
	for {
		line, err := reader.ReadString('\n')
		
		if err != nil {
			log.Printf("[%s] Error reading from serial port: %v\n", p.UniqueName, err)
			p.setState(STATE_DISCONNECTED)
			p.readChannel <- nil
			
			port.Close()
			
			p.scheduleReconnection()
			break
		}
		
		log.Printf("[%s] Read line: %s\n", p.UniqueName, line)
		
		if line == "start" {
			log.Printf("[%s] Printer restart detected\n")
			p.readChannel = nil
		} else {
			p.readChannel <- &line
		}
	}
}

func (p *Printer) scheduleReconnection() {
	go func() {
		if (p.waitBeforeReconnect()) {
			p.Start()
		}
	}()
}

func gcodeChecksum(line string) uint {
	var cs uint
	
	for _, b := range []byte(line) {
		cs ^= uint(b)
	}
	
	return cs & 0xff
}

func (p *Printer) SendCommand(command string, callback func(reply []string, err error)) {
	if p.state != STATE_CONNECTED {
		// Report error
		callback(nil, errors.New("Printer is not connected"))
		return
	}
	
	// We can only be executing a single command
	p.sendWaitGroup.Wait()
	defer p.sendWaitGroup.Done()
	
	var line *string
	if p.nextLineNo >= MAX_LINENO {
		// Reset the line counter
		p.port.WriteString("M110 N0\n")
		
		line = <-p.readChannel
		
		if line == nil {
			callback(nil, errors.New("Comm failure"))
			return
		}
		
		p.nextLineNo = 1
	}

	// Do sending

	cmd := strings.SplitN(command, " ", 1)[0]
	
	useLineNumber := cmd != "M110"
	
	if useLineNumber {
		command = fmt.Sprintf("N%d %s *%u\n", p.nextLineNo, command, gcodeChecksum(command))
		p.nextLineNo++
	} else {
		command = command + "\n"
	}

Resend:
	if p.state != STATE_CONNECTED {
		// Report error
		callback(nil, errors.New("Printer is not connected"))
		return
	}
	p.port.WriteString(command)
	
	replyLines := make([]string, 0)
	for {
		line = <- p.readChannel
		
		if line == nil {
			callback(nil, errors.New("Comm failure"))
			return
		} else if strings.HasPrefix(*line, "Resend:") {
			// Handle resend
			lineNo, _ := strconv.ParseInt((*line)[7:], 10, 0) 
			
			if int(lineNo) == (p.nextLineNo-1) {
				goto Resend
			} else {
				log.Printf("[%s] Cannot handle resend of line %d\n", p.UniqueName, int(lineNo))
				callback(nil, errors.New("Cannot handle resend of requested line"))
				return
			}
		} else {
			replyLines = append(replyLines, *line)
			
			if *line == "ok" || strings.HasPrefix(*line, "ok ") {
				callback(replyLines, nil)
				break
			} else {
				if cmd == "M190" || cmd == "M109" {
					p.parseTemperatures(cmd, *line)
				}
			}
		}
	}
}

func (p *Printer) parseTemperatures(cmd string, line string) {
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
