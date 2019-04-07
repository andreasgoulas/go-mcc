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

package gomcc

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

const (
	UpdateInterval    = 100 * time.Millisecond
	HeartbeatInterval = 45 * time.Second
	SaveInterval      = 5 * time.Minute
)

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
	Config    *Config
	MainLevel *Level
	URL       string

	playerCount int32
	salt        [16]byte

	commands     map[string]*Command
	commandsLock sync.RWMutex

	handlers     map[EventType][]EventHandler
	handlersLock sync.RWMutex

	storage    LevelStorage
	levels     []*Level
	levelsLock sync.RWMutex

	entities     []*Entity
	entitiesLock sync.RWMutex

	clients     []*Client
	clientsLock sync.RWMutex

	listener net.Listener
	stopChan chan bool

	updateTicker    *time.Ticker
	heartbeatTicker *time.Ticker
	saveTicker      *time.Ticker
}

func NewServer(config *Config, storage LevelStorage) *Server {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: config.Port})
	if err != nil {
		fmt.Printf("Server Error: %s\n", err.Error())
		return nil
	}

	server := &Server{
		Config:   config,
		commands: make(map[string]*Command),
		handlers: make(map[EventType][]EventHandler),
		storage:  storage,
		levels:   []*Level{},
		entities: []*Entity{},
		clients:  []*Client{},
		listener: listener,
		stopChan: make(chan bool),
	}

	server.generateSalt()

	mainLevel, err := server.LoadLevel(config.MainLevel)
	if err != nil {
		fmt.Printf("Server Error: Main level not found.\n")

		mainLevel = NewLevel(config.MainLevel, 128, 64, 128)
		if mainLevel == nil {
			return nil
		}

		Generators["flat"]().Generate(mainLevel)
		server.AddLevel(mainLevel)
	}

	server.MainLevel = mainLevel
	return server
}

func (server *Server) Run(wg *sync.WaitGroup) {
	wg.Add(1)

	server.updateTicker = time.NewTicker(UpdateInterval)
	go func() {
		for range server.updateTicker.C {
			server.ForEachEntity(func(entity *Entity) {
				entity.update()
			})
		}
	}()

	if SaveInterval > 0 {
		server.saveTicker = time.NewTicker(SaveInterval)
		go func() {
			for range server.saveTicker.C {
				server.ForEachLevel(func(level *Level) {
					server.SaveLevel(level)
				})
			}
		}()
	}

	if HeartbeatInterval > 0 && len(server.Config.Heartbeat) > 0 {
		server.heartbeatTicker = time.NewTicker(HeartbeatInterval)
		go func() {
			server.sendHeartbeat()
			for range server.heartbeatTicker.C {
				server.sendHeartbeat()
			}
		}()
	}

	for {
		select {
		case <-server.stopChan:
			server.clientsLock.RLock()
			clients := make([]*Client, len(server.clients))
			copy(clients, server.clients)
			server.clientsLock.RUnlock()

			for _, client := range clients {
				client.Kick("Server shutting down!")
			}

			server.updateTicker.Stop()
			server.saveTicker.Stop()
			if server.heartbeatTicker != nil {
				server.heartbeatTicker.Stop()
			}

			server.UnloadAll()
			wg.Done()
			return

		default:
			conn, err := server.listener.Accept()
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

			go client.handle()
		}
	}
}

func (server *Server) Stop() {
	server.listener.Close()
	server.stopChan <- true
}

func (server *Server) BroadcastMessage(message string) {
	fmt.Printf("%s\n", message)
	server.ForEachClient(func(client *Client) {
		client.SendMessage(message)
	})
}

func (server *Server) AddLevel(level *Level) {
	if level.server != nil {
		return
	}

	server.levelsLock.Lock()
	server.levels = append(server.levels, level)
	server.levelsLock.Unlock()

	level.server = server

	event := EventLevelLoad{level}
	server.FireEvent(EventTypeLevelLoad, &event)
}

func (server *Server) RemoveLevel(level *Level) {
	if level.server != server {
		return
	}

	server.levelsLock.Lock()
	defer server.levelsLock.Unlock()

	index := -1
	for i, l := range server.levels {
		if l == level {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	if server.MainLevel == level {
		server.MainLevel = nil
	}

	level.ForEachClient(func(client *Client) {
		client.entity.TeleportLevel(server.MainLevel)
	})

	level.server = nil

	server.levels[index] = server.levels[len(server.levels)-1]
	server.levels[len(server.levels)-1] = nil
	server.levels = server.levels[:len(server.levels)-1]

	event := EventLevelUnload{level}
	server.FireEvent(EventTypeLevelUnload, &event)
}

func (server *Server) FindLevel(name string) *Level {
	server.levelsLock.RLock()
	defer server.levelsLock.RUnlock()

	for _, level := range server.levels {
		if level.name == name {
			return level
		}
	}

	return nil
}

func (server *Server) ForEachLevel(fn func(*Level)) {
	server.levelsLock.RLock()
	for _, level := range server.levels {
		fn(level)
	}
	server.levelsLock.RUnlock()
}

func (server *Server) LoadLevel(name string) (*Level, error) {
	level := server.FindLevel(name)
	if level != nil {
		return level, nil
	}

	if server.storage == nil {
		return nil, errors.New("server: no level storage")
	}

	level, err := server.storage.Load(name)
	if err != nil {
		return nil, err
	}

	server.AddLevel(level)
	return level, nil
}

func (server *Server) SaveLevel(level *Level) {
	if server.storage == nil {
		return
	}

	event := EventLevelSave{level}
	server.FireEvent(EventTypeLevelSave, &event)

	err := server.storage.Save(level)
	if err != nil {
		fmt.Printf("Server Error: %s\n", err.Error())
	}
}

func (server *Server) UnloadLevel(level *Level) {
	server.SaveLevel(level)
	server.RemoveLevel(level)
}

func (server *Server) UnloadAll() {
	server.levelsLock.Lock()
	levels := make([]*Level, len(server.levels))
	copy(levels, server.levels)
	server.levelsLock.Unlock()

	for _, level := range levels {
		server.UnloadLevel(level)
	}
}

func (server *Server) AddEntity(entity *Entity) bool {
	server.entitiesLock.Lock()
	defer server.entitiesLock.Unlock()

	entity.id = server.generateID()
	if entity.id == 0xff {
		return false
	}

	server.entities = append(server.entities, entity)
	server.ForEachClient(func(client *Client) {
		client.sendAddPlayerList(entity)
	})

	return true
}

func (server *Server) RemoveEntity(entity *Entity) {
	server.entitiesLock.Lock()
	defer server.entitiesLock.Unlock()

	index := -1
	for i, e := range server.entities {
		if e == entity {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	server.entities[index] = server.entities[len(server.entities)-1]
	server.entities[len(server.entities)-1] = nil
	server.entities = server.entities[:len(server.entities)-1]

	server.ForEachClient(func(client *Client) {
		client.sendRemovePlayerList(entity)
	})
}

func (server *Server) FindEntity(name string) *Entity {
	server.entitiesLock.RLock()
	defer server.entitiesLock.RUnlock()

	for _, entity := range server.entities {
		if entity.name == name {
			return entity
		}
	}

	return nil
}

func (server *Server) FindEntityByID(id byte) *Entity {
	server.entitiesLock.RLock()
	defer server.entitiesLock.RUnlock()

	for _, entity := range server.entities {
		if entity.id == id {
			return entity
		}
	}

	return nil
}

func (server *Server) ForEachEntity(fn func(*Entity)) {
	server.entitiesLock.RLock()
	for _, entity := range server.entities {
		fn(entity)
	}
	server.entitiesLock.RUnlock()
}

func (server *Server) AddClient(client *Client) {
	server.clientsLock.Lock()
	server.clients = append(server.clients, client)
	server.clientsLock.Unlock()
}

func (server *Server) RemoveClient(client *Client) {
	server.clientsLock.Lock()
	defer server.clientsLock.Unlock()

	index := -1
	for i, p := range server.clients {
		if p == client {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	server.clients[index] = server.clients[len(server.clients)-1]
	server.clients[len(server.clients)-1] = nil
	server.clients = server.clients[:len(server.clients)-1]
}

func (server *Server) FindClient(name string) *Client {
	server.clientsLock.RLock()
	defer server.clientsLock.RUnlock()

	for _, client := range server.clients {
		if client.entity != nil && client.entity.name == name {
			return client
		}
	}

	return nil
}

func (server *Server) ForEachClient(fn func(*Client)) {
	server.clientsLock.RLock()
	for _, client := range server.clients {
		fn(client)
	}
	server.clientsLock.RUnlock()
}

func (server *Server) RegisterCommand(command *Command) {
	server.commandsLock.Lock()
	server.commands[command.Name] = command
	server.commandsLock.Unlock()
}

func (server *Server) ForEachCommand(fn func(*Command)) {
	server.commandsLock.RLock()
	for _, command := range server.commands {
		fn(command)
	}
	server.commandsLock.RUnlock()
}

func (server *Server) ExecuteCommand(sender CommandSender, message string) {
	args := strings.SplitN(message, " ", 2)
	if len(args) == 0 {
		return
	}

	server.commandsLock.RLock()
	command := server.commands[args[0]]
	server.commandsLock.RUnlock()

	if command == nil {
		sender.SendMessage("Unknown command!")
		return
	}

	if !sender.HasPermission(command.Permission) {
		sender.SendMessage("You do not have permission to execute this command!")
		return
	}

	if len(args) == 2 {
		message = args[1]
	} else {
		message = ""
	}

	go command.Handler(sender, command, message)
}

func (server *Server) RegisterHandler(eventType EventType, handler EventHandler) {
	server.handlersLock.Lock()
	server.handlers[eventType] = append(server.handlers[eventType], handler)
	server.handlersLock.Unlock()
}

func (server *Server) FireEvent(eventType EventType, event interface{}) {
	server.handlersLock.RLock()
	for _, handler := range server.handlers[eventType] {
		handler(eventType, event)
	}
	server.handlersLock.RUnlock()
}

func (server *Server) generateSalt() {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789"
	for i := range server.salt {
		server.salt[i] = charset[rand.Intn(len(charset))]
	}
}

func (server *Server) generateID() byte {
	for id := byte(0); id < 0xff; id++ {
		free := true
		for _, entity := range server.entities {
			if entity.id == id {
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

func (server *Server) sendHeartbeat() {
	form := url.Values{}
	form.Add("name", server.Config.Name)
	form.Add("port", strconv.Itoa(server.Config.Port))
	form.Add("max", strconv.Itoa(server.Config.MaxPlayers))
	form.Add("users", strconv.Itoa(int(server.playerCount)))
	form.Add("salt", string(server.salt[:]))
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
