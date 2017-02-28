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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ServerSoftware = "Go-MCC"

type Config struct {
	Port       int    `json:"server-port"`
	Name       string `json:"server-name"`
	MOTD       string `json:"motd"`
	Verify     bool   `json:"verify-names"`
	Public     bool   `json:"public"`
	MaxPlayers int    `json:"max-players"`
	Heartbeat  string `json:"heartbeat,omitempty"`
	MainLevel  string `json:"main-level"`
}

type Server struct {
	Config      *Config
	PlayerCount int32

	Commands     map[string]*Command
	CommandsLock sync.RWMutex

	Handlers     map[EventType][]EventHandler
	HandlersLock sync.RWMutex

	Storage    LevelStorage
	Levels     []*Level
	LevelsLock sync.RWMutex

	Entities     []*Entity
	EntitiesLock sync.RWMutex

	Clients     []*Client
	ClientsLock sync.RWMutex

	URL  string
	Salt [16]byte

	Listener net.Listener
	StopChan chan bool

	UpdateTicker    *time.Ticker
	HeartbeatTicker *time.Ticker
	SaveTicker      *time.Ticker
}

func NewServer(config *Config, storage LevelStorage) *Server {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: config.Port})
	if err != nil {
		fmt.Printf("Server Error: %s\n", err.Error())
		return nil
	}

	server := &Server{
		Config:   config,
		Commands: make(map[string]*Command),
		Handlers: make(map[EventType][]EventHandler),
		Storage:  storage,
		Levels:   []*Level{},
		Entities: []*Entity{},
		Clients:  []*Client{},
		Listener: listener,
		StopChan: make(chan bool),
	}

	server.GenerateSalt()

	mainLevel, err := server.LoadLevel(config.MainLevel)
	if err != nil {
		fmt.Printf("Server Error: Main level not found.\n")
		mainLevel = NewLevel(config.MainLevel, 128, 64, 128)
		if mainLevel == nil {
			return nil
		}

		Generators["flat"].Generate(mainLevel)
	}

	server.AddLevel(mainLevel)
	return server
}

func (server *Server) Run(wg *sync.WaitGroup) {
	wg.Add(1)

	server.UpdateTicker = time.NewTicker(100 * time.Millisecond)
	go func() {
		for range server.UpdateTicker.C {
			server.ForEachEntity(func(entity *Entity) {
				entity.Update(100 * time.Millisecond)
			})
		}
	}()

	server.SaveTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for range server.SaveTicker.C {
			server.SaveAll()
		}
	}()

	if len(server.Config.Heartbeat) > 0 {
		server.HeartbeatTicker = time.NewTicker(45 * time.Second)
		go func() {
			server.SendHeartbeat()
			for range server.HeartbeatTicker.C {
				server.SendHeartbeat()
			}
		}()
	}

	for {
		select {
		case <-server.StopChan:
			server.UpdateTicker.Stop()
			server.SaveTicker.Stop()
			if server.HeartbeatTicker != nil {
				server.HeartbeatTicker.Stop()
			}

			server.UnloadAll()
			wg.Done()
			return

		default:
			conn, err := server.Listener.Accept()
			if err != nil {
				continue
			}

			client := NewClient(conn, server)

			event := EventClientConnect{client, false}
			server.FireEvent(EventTypeClientConnect, &event)
			if event.Cancel {
				client.Disconnect()
				continue
			}

			go client.Handle()
		}
	}
}

func (server *Server) Stop() {
	server.Listener.Close()
	server.StopChan <- true
}

func (server *Server) GenerateSalt() {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789"
	for i := range server.Salt {
		server.Salt[i] = charset[rand.Intn(len(charset))]
	}
}

func (server *Server) GenerateID() byte {
	for id := byte(0); id < 0xff; id++ {
		free := true
		for _, entity := range server.Entities {
			if entity.NameID == id {
				free = false
				break
			}
		}

		if free {
			return id
		}
	}

	return 0xff
}

func (server *Server) SendHeartbeat() {
	form := url.Values{}
	form.Add("name", server.Config.Name)
	form.Add("port", strconv.Itoa(server.Config.Port))
	form.Add("max", strconv.Itoa(server.Config.MaxPlayers))
	form.Add("users", strconv.Itoa(int(server.PlayerCount)))
	form.Add("salt", string(server.Salt[:]))
	form.Add("version", "7")
	form.Add("software", ServerSoftware)
	if server.Config.Public {
		form.Add("public", "True")
	} else {
		form.Add("public", "False")
	}

	response, err := http.PostForm(server.Config.Heartbeat, form)
	if err != nil {
		fmt.Printf("Heartbeat Error: %s\n", err.Error())
		return
	}

	if response.StatusCode != 200 {
		fmt.Printf("Heartbeat Error: %s\n", response.Status)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Heartbeat Error: %s\n", err.Error())
		return
	}

	data := struct {
		Status   string     `json:"status"`
		Response string     `json:"response"`
		Errors   [][]string `json:"errors"`
	}{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return
	}

	if len(data.Errors) > 0 && len(data.Errors[0]) > 0 {
		fmt.Printf("Heartbeat Error: %s\n", data.Errors[0][0])
		return
	}

	server.URL = string(body)
}

func (server *Server) BroadcastMessage(message string) {
	fmt.Printf("%s\n", message)
	server.ForEachClient(func(client *Client) {
		client.SendMessage(message)
	})
}

func (server *Server) AddLevel(level *Level) {
	if level.Server != nil {
		return
	}

	server.LevelsLock.Lock()
	server.Levels = append(server.Levels, level)
	server.LevelsLock.Unlock()

	level.Server = server

	event := EventLevelLoad{level}
	server.FireEvent(EventTypeLevelLoad, &event)
}

func (server *Server) RemoveLevel(level *Level) {
	if level.Server != server {
		return
	}

	server.LevelsLock.Lock()
	defer server.LevelsLock.Unlock()

	index := -1
	for i, l := range server.Levels {
		if l == level {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	level.Server = server

	server.Levels[index] = server.Levels[len(server.Levels)-1]
	server.Levels[len(server.Levels)-1] = nil
	server.Levels = server.Levels[:len(server.Levels)-1]

	event := EventLevelUnload{level}
	server.FireEvent(EventTypeLevelUnload, &event)
}

func (server *Server) FindLevel(name string) *Level {
	server.LevelsLock.RLock()
	defer server.LevelsLock.RUnlock()

	for _, level := range server.Levels {
		if level.Name == name {
			return level
		}
	}

	return nil
}

func (server *Server) ForEachLevel(fn func(*Level)) {
	server.LevelsLock.RLock()
	for _, level := range server.Levels {
		fn(level)
	}
	server.LevelsLock.RUnlock()
}

func (server *Server) LoadLevel(name string) (*Level, error) {
	level := server.FindLevel(name)
	if level != nil {
		return level, nil
	}

	if server.Storage == nil {
		return nil, errors.New("server: no level storage")
	}

	level, err := server.Storage.Load(name)
	if err != nil {
		return nil, err
	}

	server.AddLevel(level)
	return level, nil
}

func (server *Server) UnloadLevel(level *Level) {
	server.ClientsLock.RLock()
	clients := make([]*Client, len(server.Clients))
	copy(clients, server.Clients)
	server.ClientsLock.RUnlock()

	for _, client := range clients {
		client.Kick("Server shutting down!")
	}

	if server.Storage != nil {
		event := EventLevelSave{level}
		server.FireEvent(EventTypeLevelSave, &event)

		err := server.Storage.Save(level)
		if err != nil {
			fmt.Printf("Server Error: %s\n", err.Error())
		}
	}

	server.RemoveLevel(level)
}

func (server *Server) UnloadAll() {
	server.LevelsLock.Lock()
	levels := make([]*Level, len(server.Levels))
	copy(levels, server.Levels)
	server.LevelsLock.Unlock()

	for _, level := range levels {
		server.UnloadLevel(level)
	}
}

func (server *Server) SaveAll() {
	if server.Storage == nil {
		return
	}

	server.ForEachLevel(func(level *Level) {
		event := EventLevelSave{level}
		server.FireEvent(EventTypeLevelSave, &event)

		err := server.Storage.Save(level)
		if err != nil {
			fmt.Printf("Server Error: %s\n", err.Error())
		}
	})
}

func (server *Server) MainLevel() *Level {
	server.LevelsLock.RLock()
	defer server.LevelsLock.RUnlock()

	if len(server.Levels) > 0 {
		return server.Levels[0]
	}

	return nil
}

func (server *Server) AddEntity(entity *Entity) byte {
	server.EntitiesLock.Lock()
	defer server.EntitiesLock.Unlock()

	entity.NameID = server.GenerateID()
	if entity.NameID == 0xff {
		return 0xff
	}

	server.Entities = append(server.Entities, entity)
	server.ForEachClient(func(client *Client) {
		client.SendAddPlayerList(entity)
	})

	return entity.NameID
}

func (server *Server) RemoveEntity(entity *Entity) {
	server.EntitiesLock.Lock()
	defer server.EntitiesLock.Unlock()

	index := -1
	for i, e := range server.Entities {
		if e == entity {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	server.Entities[index] = server.Entities[len(server.Entities)-1]
	server.Entities[len(server.Entities)-1] = nil
	server.Entities = server.Entities[:len(server.Entities)-1]

	server.ForEachClient(func(client *Client) {
		client.SendRemovePlayerList(entity)
	})
}

func (server *Server) FindEntity(name string) *Entity {
	server.EntitiesLock.RLock()
	defer server.EntitiesLock.RUnlock()

	for _, entity := range server.Entities {
		if entity.Name == name {
			return entity
		}
	}

	return nil
}

func (server *Server) ForEachEntity(fn func(*Entity)) {
	server.EntitiesLock.RLock()
	for _, entity := range server.Entities {
		fn(entity)
	}
	server.EntitiesLock.RUnlock()
}

func (server *Server) AddClient(client *Client) {
	server.ClientsLock.Lock()
	server.Clients = append(server.Clients, client)
	server.ClientsLock.Unlock()
}

func (server *Server) RemoveClient(client *Client) {
	server.ClientsLock.Lock()
	defer server.ClientsLock.Unlock()

	index := -1
	for i, p := range server.Clients {
		if p == client {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	server.Clients[index] = server.Clients[len(server.Clients)-1]
	server.Clients[len(server.Clients)-1] = nil
	server.Clients = server.Clients[:len(server.Clients)-1]
}

func (server *Server) ForEachClient(fn func(*Client)) {
	server.ClientsLock.RLock()
	for _, client := range server.Clients {
		fn(client)
	}
	server.ClientsLock.RUnlock()
}

func (server *Server) RegisterCommand(command *Command) {
	server.CommandsLock.Lock()
	server.Commands[command.Name] = command
	server.CommandsLock.Unlock()
}

func (server *Server) ExecuteCommand(sender CommandSender, message string) {
	args := strings.Fields(message)
	if len(args) == 0 {
		return
	}

	server.CommandsLock.RLock()
	command := server.Commands[args[0]]
	server.CommandsLock.RUnlock()

	if command == nil {
		sender.SendMessage("Unknown command!")
		return
	}

	go command.Handler.HandleCommand(sender, command, args[1:])
}

func (server *Server) RegisterHandler(eventType EventType, handler EventHandler) {
	server.HandlersLock.Lock()
	server.Handlers[eventType] = append(server.Handlers[eventType], handler)
	server.HandlersLock.Unlock()
}

func (server *Server) FireEvent(eventType EventType, event interface{}) {
	server.HandlersLock.RLock()
	for _, handler := range server.Handlers[eventType] {
		handler.Handle(eventType, event)
	}
	server.HandlersLock.RUnlock()
}
