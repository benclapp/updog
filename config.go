package main

import (
	"io/ioutil"
	"log"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"
)

var config = conf{}

var (
	timeout       = kingpin.Flag("timeout", "Timeout for dependency checks").Short('t').Default("5s").Duration()
	configPath    = kingpin.Flag("config.path", "Path of configuration file").Short('c').Default("updog.yaml").String()
	listenAddress = kingpin.Flag("listen.address", "Address to listen on for HTTP requests").Default(":1111").String()
)

var (
	depTimeout time.Duration
	addr       string
)

func configure() {

	if configPath == nil {
		logger.Log("msg", "Config file required")
	}

	if timeout == nil {
		logger.Log("msg", "timeout required")
	}

	if listenAddress == nil {
		logger.Log("msg", "listen.address required")
	}

	cfg := *configPath
	config.getConf(cfg)

	depTimeout = *timeout

	addr = *listenAddress
}

func (c *conf) getConf(path string) *conf {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

type conf struct {
	Dependencies struct {
		HTTP []struct {
			Name         string `yaml:"name"`
			HTTPEndpoint string `yaml:"http_endpoint"`
		} `yaml:"http"`
		Redis []struct {
			Name     string `yaml:"name"`
			Address  string `yaml:"address"`
			Password string `yaml:"password"`
			Ssl      bool   `yaml:"ssl"`
		} `yaml:"redis"`
		SQL []struct {
			Name             string `yaml:"name"`
			ConnectionString string `yaml:"connectionString"`
			Type             string `yaml:"type"`
		} `yaml:"sql"`
	} `yaml:"dependencies"`
}
