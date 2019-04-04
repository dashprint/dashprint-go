package main

import (
        "github.com/jochenvg/go-udev"
//		"fmt"
		"strings"
		"regexp"
		"strconv"
)

type DiscoveredPrinter struct {
	DevicePath string `json:"path"`
	DeviceName string `json:"name"`
	DeviceVendor string `json:"vendor"`
	DeviceSerial string `json:"serial"`
}

func UdevPrinterDiscovery() []DiscoveredPrinter {
	u := udev.Udev{}
	e := u.NewEnumerate()

	e.AddMatchSubsystem("tty")
	e.AddMatchProperty("ID_BUS", "usb")

	devices, _ := e.Devices()

	printers := make([]DiscoveredPrinter, len(devices))
	for i := range devices {
			device := devices[i]

			printers[i] = DiscoveredPrinter{
				DeviceName: unescape(device.PropertyValue("ID_MODEL_ENC")),
				DeviceVendor: unescape(device.PropertyValue("ID_VENDOR_ENC")),
				DeviceSerial: device.PropertyValue("ID_SERIAL"),
				DevicePath: chooseDeviceLink(device.PropertyValue("DEVLINKS")),
			}
	}
	return printers
}

func chooseDeviceLink(devlinks string) string {
	var bestlink string;

	for index, link := range strings.Split(devlinks, " ") {
		if index == 0 || strings.HasPrefix(link, "/dev/serial/by-id/") {
			bestlink = link;
		}
	}

	return bestlink;
}

func unescape(text string) string {
	re := regexp.MustCompile(`\\x(.{2})`)
	return ReplaceAllStringSubmatchFunc(re, text, func (groups []string) string {
		c, _ := strconv.ParseInt(groups[1], 16, 8)
		return string(c)
	})
}

func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}


