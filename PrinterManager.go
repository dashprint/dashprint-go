package main

import (
	"github.com/gosimple/slug"
	"fmt"
)

var printers map[string]*Printer

func loadPrinters(config Configuration) {
	for _, ps := range config.printers {
		printer := LoadPrinter(ps)
		printers[printer.UniqueName] = printer

		if !printer.Stopped {
			printer.Start()
		}
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
	printers[printer.UniqueName] = printer
	
	if !ps.Stopped {
		printer.Start()
	}
	
	return printer.UniqueName
}

