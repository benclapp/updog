package config

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"
)

var logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

var (
	version       string
	configPath    = kingpin.Flag("config.path", "Path of configuration file").Short('c').String()
	listenAddress = kingpin.Flag("listen.address", "Address to listen on for HTTP requests").Default(":1111").String()
	timeout       = kingpin.Flag("timeout", "Timeout for dependency checks").Short('t').Default("5s").Duration()
)

type Configurable interface {
	GetAddress() string
	GetTimeout() time.Duration
	GetDependencies() Dependencies
}

type Dependencies struct {
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
}

//FIXME this is just a hack holder, what we are reading from file are dependencies only
type Configuration struct {
	Address      string        `yaml:"address"`
	Timeout      time.Duration `yaml:"timeout"`
	Dependencies Dependencies  `yaml:"dependencies"`
}

func NewConfig() *Configuration {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(version)
	kingpin.CommandLine.Help = "Service to aggregate health checks. Returns 502 if any fail."
	kingpin.Parse()

	if configPath == nil {
		logger.Log("msg", "Config file required")
		os.Exit(1)
	}

	logger.Log("msg", "Config file", *configPath)

	config := &Configuration{}
	readFile(config, *configPath)

	if config.Address == "" {
		logger.Log("msg", "using default listen address")
		config.Address = *listenAddress
	}

	if config.Timeout == 0 {
		logger.Log("msg", "using default timeout")
		config.Timeout = *timeout
	}

	return config
}

func readFile(cfg *Configuration, path string) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		level.Error(logger).Log("yamlFile.Get err #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, cfg)
	if err != nil {
		level.Error(logger).Log("Unmarshal: %v", err)
	}
}

func (c Configuration) GetDependencies() Dependencies {
	return c.Dependencies
}

func (c Configuration) GetAddress() string {
	return c.Address
}

func (c Configuration) GetTimeout() time.Duration {
	return c.Timeout
}
