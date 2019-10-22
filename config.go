package main

import (
	"errors"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Bind     string   `yaml:"bind"`
	Zone     string   `yaml:"zone"`
	Services []string `yaml:"services"`
}

func validateConfig(c *Config) error {
	if c.Bind == "" {
		return errors.New("'Bind' is not set")
	}

	if c.Zone == "" {
		return errors.New("'Zone' is not set")
	}

	if len(c.Services) == 0 {
		return errors.New("'Services' must be defined")
	}

	return nil
}

func LoadConfig(path string) (*Config, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		return nil, err
	}

	err = validateConfig(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
