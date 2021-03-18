package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"plugin"
	"sync"

	"github.com/AndreasGoulas/go-mcc/mcc"
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
