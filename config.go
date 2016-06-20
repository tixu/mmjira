package main

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

//InstanceConfig is used to configure the bridge.
type InstanceConfig struct {
	Host       string
	Port       int
	Metrics    bool
	Debug      bool
	DumpDir    string
	Hooks      map[string]string
	MMicon     string
	MMuser     string
	Profile    string
	ProfileDir string
}

// UnmarshalYAML parse the configuration file.
func (c *InstanceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var aux struct {
		Hostname   string            `yaml:"host"`
		Port       string            `yaml:"port"`
		Metrics    string            `yaml:"metrics"`
		Debug      string            `yaml:"debug"`
		Hooks      map[string]string `yaml:"hooks"`
		DumpDir    string            `yaml:"dumpdir"`
		MMuser     string            `yaml:"mmuser"`
		MMIcon     string            `yaml:"mmicon"`
		Profile    string            `yaml:"profile"`
		ProfileDir string            `yaml:"profiledir"`
	}
	log.Println("validating config")
	if err := unmarshal(&aux); err != nil {
		return err
	}
	if aux.Hostname == "" {
		return errors.New("Brigge config: invalid `hostname`")
	}

	port, err := strconv.Atoi(aux.Port)
	if err != nil {
		return errors.New("Bridge config: invalid `port`")
	}

	// Test Kitchen stores the port as an string
	metrics, err := strconv.ParseBool(aux.Metrics)
	if err != nil {
		return errors.New("Bridge config: invalid `metrics`")
	}
	debug, err := strconv.ParseBool(aux.Debug)
	if err != nil {
		return errors.New("Bridge config: invalid `debug`")
	}
	upperHooks := make(map[string]string)
	for key, value := range aux.Hooks {
		upperHooks[strings.ToUpper(key)] = value
	}
	c.Host = aux.Hostname
	c.Port = port
	c.Metrics = metrics
	c.Hooks = upperHooks
	c.Debug = debug

	c.DumpDir = aux.DumpDir
	c.MMicon = aux.MMIcon
	c.MMuser = aux.MMuser
	c.Profile = aux.Profile
	c.ProfileDir = aux.ProfileDir
	log.Println("config validated")
	return nil
}
