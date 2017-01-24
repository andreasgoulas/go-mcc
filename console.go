// Copyright 2017 Andrew Goulas
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
	"fmt"
	"os"
	"os/signal"
	"sync"
)

type Console struct {
	Server    *Server
	WaitGroup *sync.WaitGroup
	Signal    chan os.Signal
}

func NewConsole(server *Server, wg *sync.WaitGroup) *Console {
	console := &Console{
		server, wg,
		make(chan os.Signal),
	}

	server.Commands["stop"] = &Command{
		"stop",
		"Stop the server.",
		console,
	}

	signal.Notify(console.Signal, os.Interrupt)
	go func() {
		signal := <-console.Signal
		if signal == os.Interrupt {
			console.Stop()
		}
	}()

	return console
}

func (console *Console) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		console.Server.ExecuteCommand(console, scanner.Text())
	}
}

func (console *Console) Stop() {
	console.Server.Stop()
	console.WaitGroup.Wait()
	os.Exit(0)
}

func (console *Console) SendMessage(message string) {
	fmt.Println(message)
}

func (console *Console) IsOperator() bool {
	return true
}

func (console *Console) HandleCommand(sender CommandSender, command *Command, args []string) {
	switch command.Name {
	case "stop":
		if !sender.IsOperator() {
			sender.SendMessage("You are not an operator!")
			return
		}

		console.Stop()
	}
}
