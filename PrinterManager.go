package main

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
