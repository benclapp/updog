package config

import (
	"testing"
    "gotest.tools/assert"
)

func TestReadConfiguration(t *testing.T) {
	configPath := "updog.test.yaml"
	cfg := &configuration{}
	readFile(cfg, configPath)
	assert.Assert(t, cfg != nil)
	assert.Assert(t, len(cfg.Dependencies.HTTP) != 0) // NotEmpty
	assert.Assert(t, len(cfg.Dependencies.Redis) != 0) // NotEmpty
	assert.Assert(t, len(cfg.Dependencies.SQL) != 0) // NotEmpty
}

func TestCreateConfigurationInterface(t *testing.T) {
	cfgObj := &configuration{}
	cfgObj.Address = "1111"
	
	var cfgInterface Configurable
	cfgInterface = cfgObj

	assert.Assert(t, cfgObj.Address == cfgInterface.GetAddress())
}