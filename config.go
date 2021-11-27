package main

import (
	"github.com/tkanos/gonfig"
	"log"
)

type Configuration struct {
	ApiKey     string
	DbFileName string
}

func GetConfig() Configuration {
	configuration := Configuration{}
	fileName := "./dev_config.json"
	err := gonfig.GetConf(fileName, &configuration)
	if err != nil {
		log.Fatalf("Error load config: %s", err)
	}

	return configuration
}
