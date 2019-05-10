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
	"encoding/json"
	"io/ioutil"
	"log"
	"plugin"
	"sync"

	"github.com/structinf/Go-MCC/gomcc"
	"github.com/structinf/Go-MCC/storage"
)

var defaultConfig = &gomcc.Config{
	Port:       25565,
	Name:       "Go-MCC",
	MOTD:       "Welcome!",
	Verify:     false,
	Public:     true,
	MaxPlayers: 32,
	Heartbeat:  "http://www.classicube.net/heartbeat.jsp",
	MainLevel:  "main",
}

func readConfig(path string) *gomcc.Config {
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
		config := &gomcc.Config{}
		err = json.Unmarshal(file, config)
		if err != nil {
			log.Printf("readConfig: %s\n", err)
			config = defaultConfig
		}

		return config
	}
}

func loadPlugins(path string, server *gomcc.Server) {
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

		initFn, ok := sym.(func() gomcc.Plugin)
		if !ok {
			continue
		}

		plug := initFn()
		server.RegisterPlugin(plug)
	}
}

func main() {
	config := readConfig("server.properties")
	lvlstorage := storage.NewLvlStorage("levels/")
	server := gomcc.NewServer(config, lvlstorage)
	if server == nil {
		return
	}

	loadPlugins("plugins/", server)

	wg := &sync.WaitGroup{}
	go server.Run(wg)

	console := NewConsole(server, wg)
	console.Run()
}
