package main

import (
	"github.com/gosimple/slug"
	"fmt"
	"sync"
)

var printers map[string]*Printer = make(map[string]*Printer)
var defaultPrinter string
var printerMutex sync.RWMutex

func loadPrinters(config Configuration) {
	defaultPrinter = ""

	for _, ps := range config.Printers {
		printer := LoadPrinter(ps)
		printers[printer.UniqueName] = printer

		if defaultPrinter == "" {
			defaultPrinter = printer.UniqueName
		}

		if !printer.Stopped {
			printer.Start()
		}
	}

	if _, ok := printers[config.Default]; ok {
		defaultPrinter = config.Default
	}
}

func addPrinter(ps PrinterSettings) string {
	// Generate a unique name if not assigned yet
	if ps.UniqueName == "" {
		uniqueName := slug.Make(ps.Name)

		if _, ok := printers[uniqueName]; ok {
			num := 2
			for {
				name := fmt.Sprintf("%s%d", uniqueName, num)

				if _, ok := printers[name]; !ok {
					uniqueName = name
					break
				}
				num++
			}
		}

		ps.UniqueName = uniqueName
	}

    printer := LoadPrinter(ps)

	printerMutex.Lock()
	printers[printer.UniqueName] = printer
	printerMutex.Unlock()

	if !ps.Stopped {
		printer.Start()
	}

	return printer.UniqueName
}

