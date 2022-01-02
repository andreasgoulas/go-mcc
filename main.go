package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"plugin"
	"sync"

	"github.com/andreasgoulas/go-mcc/mcc"
)

var defaultConfig = &mcc.Config{
	Port:       25565,
	Name:       "Go-MCC",
	MOTD:       "Welcome!",
	Verify:     false,
	Public:     true,
	MaxPlayers: 32,
	Heartbeat:  "http://www.classicube.net/heartbeat.jsp",
	MainLevel:  "main",
}

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

func readConfig(path string) *mcc.Config {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		data, err := json.MarshalIndent(defaultConfig, "", "\t")
		if err != nil {
			log.Printf("readConfig: %s\n", err)
			return defaultConfig
		}

		if err := ioutil.WriteFile(path, data, 0644); err != nil {
			log.Printf("readConfig: %s\n", err)
		}

		return defaultConfig
	} else {
		config := &mcc.Config{}
		err = json.Unmarshal(file, config)
		if err != nil {
			log.Printf("readConfig: %s\n", err)
			config = defaultConfig
		}

		return config
	}
}

func loadPlugins(path string, server *mcc.Server) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		lib, err := plugin.Open(path + file.Name())
		if err != nil {
			log.Printf("loadPlugins: %s\n", err)
			continue
		}

		sym, err := lib.Lookup("Initialize")
		if err != nil {
			log.Printf("loadPlugins: %s\n", err)
			continue
		}

		initFn, ok := sym.(func() mcc.Plugin)
		if !ok {
			continue
		}

		plug := initFn()
		server.AddPlugin(plug)
	}
}

func main() {
	config := readConfig("server.json")
	cwstorage := mcc.NewCwStorage("levels/")
	server := mcc.NewServer(config, cwstorage)
	if server == nil {
		return
	}

	loadPlugins("plugins/", server)

	var wg sync.WaitGroup
	if err := server.Start(&wg); err != nil {
		log.Println(err)
		return
	}

	console := newConsole(server, &wg)
	console.run()
}
