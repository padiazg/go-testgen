package main

import "fmt"

type Config struct {
	Url string
}

type Engine struct {
	config *Config
	active bool
}

func New(cfg *Config) *Engine {
	return &Engine{config: cfg, active: false}
}

func (e *Engine) Start() error {
	e.active = true
	return nil
}

func (e *Engine) Notify(msg string) error {
	if !e.active {
		return fmt.Errorf("engine not active")
	}
	return nil
}

func SendNotification(msg string) error {
	return nil
}
