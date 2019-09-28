// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/structinf/Go-MCC/gomcc"
)

const (
	PermOperator = 1 << 0
)

type Console struct {
	server    *gomcc.Server
	waitGroup *sync.WaitGroup
	signal    chan os.Signal
}

func NewConsole(server *gomcc.Server, waitGroup *sync.WaitGroup) *Console {
	console := &Console{
		server,
		waitGroup,
		make(chan os.Signal),
	}

	server.RegisterCommand(&gomcc.Command{
		Name:        "stop",
		Description: "Stop the server.",
		Permissions: PermOperator,
		Handler:     console.handleStop,
	})

	signal.Notify(console.signal, os.Interrupt)
	go func() {
		signal := <-console.signal
		if signal == os.Interrupt {
			console.Stop()
		}
	}()

	return console
}

func (console *Console) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		console.server.ExecuteCommand(console, scanner.Text())
	}
}

func (console *Console) Stop() {
	console.server.Stop()
	console.waitGroup.Wait()
	os.Exit(0)
}

func (console *Console) Server() *gomcc.Server {
	return console.server
}

func (console *Console) Name() string {
	return "Console"
}

func (console *Console) SendMessage(message string) {
	log.Println(message)
}

// HasPermission implements CommandSender.
func (console *Console) HasPermission(command *gomcc.Command) bool {
	return true
}

func (console *Console) handleStop(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	console.Stop()
}
