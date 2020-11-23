package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/structinf/go-mcc/mcc"
)

const (
	PermOperator = 1 << 0
)

type console struct {
	server    *mcc.Server
	waitGroup *sync.WaitGroup
	signal    chan os.Signal
}

func newConsole(server *mcc.Server, waitGroup *sync.WaitGroup) *console {
	console := &console{
		server,
		waitGroup,
		make(chan os.Signal),
	}

	server.AddCommand(&mcc.Command{
		Name:        "stop",
		Description: "Stop the server.",
		Usage:       "/stop",
		Permissions: PermOperator,
		Handler:     console.handleStop,
	})

	signal.Notify(console.signal, os.Interrupt)
	go func() {
		signal := <-console.signal
		if signal == os.Interrupt {
			console.stop()
		}
	}()

	return console
}

func (console *console) run() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		console.server.ExecuteCommand(console, scanner.Text())
	}
}

func (console *console) stop() {
	console.server.Stop()
	console.waitGroup.Wait()
	os.Exit(0)
}

// Server implements mcc.CommandSender.
func (console *console) Server() *mcc.Server {
	return console.server
}

// Name implements mcc.CommandSender.
func (console *console) Name() string {
	return "Console"
}

// SendMessage implements mcc.CommandSender.
func (console *console) SendMessage(message string) {
	log.Println(message)
}

// CanExecute implements mcc.CommandSender.
func (console *console) CanExecute(command *mcc.Command) bool {
	return true
}

func (console *console) handleStop(sender mcc.CommandSender, command *mcc.Command, message string) {
	console.stop()
}
