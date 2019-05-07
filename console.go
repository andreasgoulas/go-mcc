// Copyright 2017-2019 Andrew Goulas
// https://www.structinf.com
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"sync"

	"Go-MCC/gomcc"
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
		"stop",
		"Stop the server.",
		"stop",
		console.handleStop,
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

func (console *Console) HasPermission(permission string) bool {
	return true
}

func (console *Console) handleStop(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	console.Stop()
}
