package main

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server struct {
		Address      string        `default:":8080" required:"true"`
		ReadTimeout  time.Duration `default:"10s" required:"true"`
		WriteTimeout time.Duration `default:"10s" required:"true"`
	}
}

func ParseConfig() (*Config, error) {
	var config Config

	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("config: failed to parse from env, %v", err)
	}

	return &config, nil
}
