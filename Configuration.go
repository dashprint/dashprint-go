package main

import (
	"github.com/spf13/viper"
	"log"
	"encoding/json"
	"io/ioutil"
	"os/user"
)

type Configuration struct {
	Printers []PrinterSettings `json:"printers"`
	Default string `json:"defaultPrinter"`
}


func loadConfig() {
	var configuration Configuration

	viper.SetConfigName("dashprint")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.local/share")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Cannot load config file: ", err)
		return
	}

	err := viper.Unmarshal(&configuration)
	if err != nil {
		log.Println("Unable to decode config file: ", err)
	}

	loadPrinters(configuration)
}

func saveConfig() {
	log.Println("Saving configuration...")
	config := Configuration{}

	config.Default = defaultPrinter
	config.Printers = make([]PrinterSettings, len(printers))

	i := 0
	for _, printer := range printers {
		config.Printers[i] = printer.PrinterSettings
		i++
	}

	b, _ := json.MarshalIndent(config, "", "  ")

	user, err := user.Current()
	if err != nil {
		log.Println("Failed to save configuration: ", err)
		return
	}
	err = ioutil.WriteFile(user.HomeDir + "/.local/share/dashprint.json", b, 0644)

	if err != nil {
		log.Println("Failed to save configuration: ", err)
	}
}
