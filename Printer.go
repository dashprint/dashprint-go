package main

const (
	STATE_STOPPED = iota
	STATE_DISCONNECTED = iota
	STATE_INITIALIZING = iota
	STATE_CONNECTED = iota
)

type PrinterSettings struct {
	Name string `json:"name"`
	DevicePath string `json:"devicePath"`
	UniqueName string `json:"uniqueName"`
	BaudRate uint `json:"baudRate"`
	Stopped bool `json:"stopped"`
	PrintArea PrintArea `json:"printArea"`
}

type Printer struct {
	PrinterSettings
	state int
}

type PrintArea struct {
	Width, Height, Depth uint
}

func LoadPrinter(settings PrinterSettings) *Printer {
	newState := STATE_STOPPED
	if !settings.Stopped {
		newState = STATE_DISCONNECTED
	}
	
	p := &Printer{settings, newState}
	return p
}


