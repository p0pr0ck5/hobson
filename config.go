package main

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type config struct {
	Bind     string   `yaml:"bind"`
	Zone     string   `yaml:"zone"`
	Services []string `yaml:"services"`
}

func validateConfig(c *config) error {
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

func loadConfig(path string) (*config, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config config
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
