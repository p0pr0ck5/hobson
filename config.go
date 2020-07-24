package main

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config details how hobson should operate
type Config struct {
	Bind     string   `yaml:"bind"`
	PromBind string   `yaml:"prometheus_bind"`
	Zone     string   `yaml:"zone"`
	Services []string `yaml:"services"`
}

func hasDuplicate(haystack []string) bool {
	m := make(map[string]bool)

	for _, s := range haystack {
		if _, found := m[s]; found == true {
			return true
		}

		m[s] = true
	}

	return false
}

// NewConfig creates and validate a new Config object, given a specified filesystem path
func NewConfig(path string) (*Config, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		return nil, err
	}

	err = config.Validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate returns an error if an invalid configuration is present in the Config
func (c *Config) Validate() error {
	if c.Bind == "" {
		return errors.New("'Bind' is not set")
	}

	if c.PromBind == "" {
		return errors.New("'PromBind' is not set")
	}

	if c.Zone == "" {
		return errors.New("'Zone' is not set")
	}

	if len(c.Services) == 0 {
		return errors.New("'Services' must be defined")
	}

	if hasDuplicate(c.Services) {
		return errors.New("'Services' contains duplicate entries")
	}

	return nil
}
