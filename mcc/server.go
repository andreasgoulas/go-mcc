// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package mcc

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ServerSoftware = "Go-MCC"

	UpdateInterval    = 50 * time.Millisecond
	HeartbeatInterval = 45 * time.Second
	SaveInterval      = 5 * time.Minute
)

// Config is used to configure a server.
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

// Plugin is the interface that must be implemented by all plugins.
type Plugin interface {
	Name() string
	Enable(*Server)
	Disable(*Server)
}

// Server represents a game server.
type Server struct {
	Config    *Config
	MainLevel *Level
	URL       string
	Colors    []ColorDesc
	Hotkeys   []HotkeyDesc

	playerCount int32
	salt        [16]byte

	commands     map[string]*Command
	commandsLock sync.RWMutex

	handlers     map[EventType][]EventHandler
	handlersLock sync.RWMutex

	generators     map[string]GeneratorFunc
	generatorsLock sync.RWMutex

	storage    LevelStorage
	levels     []*Level
	levelsLock sync.RWMutex

	entities     []*Entity
	entitiesLock sync.RWMutex

	players     []*Player
	playersLock sync.RWMutex

	plugins     []Plugin
	pluginsLock sync.RWMutex

	listener net.Listener
	stopChan chan bool

	updateTicker    *time.Ticker
	heartbeatTicker *time.Ticker
	saveTicker      *time.Ticker
}

// NewServer returns a new Server.
func NewServer(config *Config, storage LevelStorage) *Server {
	server := &Server{
		Config:     config,
		commands:   make(map[string]*Command),
		handlers:   make(map[EventType][]EventHandler),
		generators: make(map[string]GeneratorFunc),
		storage:    storage,
		stopChan:   make(chan bool),
	}

	server.generateSalt()

	server.generators["flat"] = NewFlatGenerator
	mainLevel, err := server.LoadLevel(config.MainLevel)
	if err != nil {
		log.Printf("Main level not found.\n")
		mainLevel = NewLevel(config.MainLevel, 128, 64, 128)
		if mainLevel == nil {
			return nil
		}

		NewFlatGenerator().Generate(mainLevel)
		server.AddLevel(mainLevel)
	}

	server.MainLevel = mainLevel
	return server
}

// Start starts the server.
// When the server is stopped, wg will be notified.
func (server *Server) Start(wg *sync.WaitGroup) (err error) {
	addr := net.TCPAddr{Port: server.Config.Port}
	if server.listener, err = net.ListenTCP("tcp", &addr); err != nil {
		return
	}

	go server.run(wg)
	return
}

// Stop stops the server, disconnects all clients, disables all plugins and
// unloads all levels.
func (server *Server) Stop() {
	server.listener.Close()
	server.stopChan <- true
}

// BroadcastMessage broadcasts a message to all players.
func (server *Server) BroadcastMessage(message string) {
	log.Printf("%s\n", message)
	server.ForEachPlayer(func(player *Player) {
		player.SendMessage(message)
	})
}

// AddLevel adds level to the server.
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

// RemoveLevel removes level from the server.
// All players in level will be moved to the main level.
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

	level.ForEachPlayer(func(player *Player) {
		player.TeleportLevel(server.MainLevel)
	})

	level.server = nil
	server.levels[index] = server.levels[len(server.levels)-1]
	server.levels[len(server.levels)-1] = nil
	server.levels = server.levels[:len(server.levels)-1]

	event := EventLevelUnload{level}
	server.FireEvent(EventTypeLevelUnload, &event)
}

// FindLevel returns the level with the specified name.
func (server *Server) FindLevel(name string) *Level {
	server.levelsLock.RLock()
	defer server.levelsLock.RUnlock()

	for _, level := range server.levels {
		if level.Name == name {
			return level
		}
	}

	return nil
}

// ForEachLevel calls fn for each level.
func (server *Server) ForEachLevel(fn func(*Level)) {
	server.levelsLock.RLock()
	for _, level := range server.levels {
		fn(level)
	}
	server.levelsLock.RUnlock()
}

// LoadLevel attempts to load the level with the specified name.
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

// SaveLevel saves level.
func (server *Server) SaveLevel(level *Level) {
	if server.storage == nil || !level.Dirty {
		return
	}

	event := EventLevelSave{level}
	server.FireEvent(EventTypeLevelSave, &event)

	err := server.storage.Save(level)
	if err != nil {
		log.Printf("SaveLevel: %s\n", err.Error())
	}
}

// UnloadLevel saves and removes level from the server.
// All players in level will be moved to the main level.
func (server *Server) UnloadLevel(level *Level) {
	server.SaveLevel(level)
	server.RemoveLevel(level)
}

// AddEntity adds entity to the server.
// It returns true on success, or false if the server is full.
func (server *Server) AddEntity(entity *Entity) bool {
	server.entitiesLock.Lock()
	defer server.entitiesLock.Unlock()

	entity.id = server.generateID()
	if entity.id == 0xff {
		return false
	}

	server.entities = append(server.entities, entity)
	server.ForEachPlayer(func(player *Player) {
		player.sendAddPlayerList(entity)
	})

	return true
}

// RemoveEntity removes entity from the server.
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

	server.ForEachPlayer(func(player *Player) {
		player.sendRemovePlayerList(entity)
	})
}

// FindEntity returns the entity with the specified name.
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

// FindEntityByID returns the entity with the specified ID.
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

// ForEachEntity calls fn for each entity.
func (server *Server) ForEachEntity(fn func(*Entity)) {
	server.entitiesLock.RLock()
	for _, entity := range server.entities {
		fn(entity)
	}
	server.entitiesLock.RUnlock()
}

// AddPlayer adds player to the server.
func (server *Server) AddPlayer(player *Player) {
	server.playersLock.Lock()
	server.players = append(server.players, player)
	server.playersLock.Unlock()
}

// RemovePlayer removes player from the server.
func (server *Server) RemovePlayer(player *Player) {
	server.playersLock.Lock()
	defer server.playersLock.Unlock()

	index := -1
	for i, p := range server.players {
		if p == player {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	server.players[index] = server.players[len(server.players)-1]
	server.players[len(server.players)-1] = nil
	server.players = server.players[:len(server.players)-1]
}

// FindPlayer returns the player with the specified name.
func (server *Server) FindPlayer(name string) *Player {
	server.playersLock.RLock()
	defer server.playersLock.RUnlock()

	for _, player := range server.players {
		if player.name == name {
			return player
		}
	}

	return nil
}

// ForEachPlayer calls fn for each player.
func (server *Server) ForEachPlayer(fn func(*Player)) {
	server.playersLock.RLock()
	for _, player := range server.players {
		fn(player)
	}
	server.playersLock.RUnlock()
}

// RegisterCommand registers the specified command.
func (server *Server) RegisterCommand(command *Command) {
	server.commandsLock.Lock()
	server.commands[command.Name] = command
	server.commandsLock.Unlock()
}

// Findcommand returns the command with the specified name.
func (server *Server) FindCommand(name string) *Command {
	server.commandsLock.RLock()
	defer server.commandsLock.RUnlock()

	for _, command := range server.commands {
		if command.Name == name {
			return command
		}
	}

	return nil
}

// ForEachCommand calls fn for each command.
func (server *Server) ForEachCommand(fn func(*Command)) {
	server.commandsLock.RLock()
	for _, command := range server.commands {
		fn(command)
	}
	server.commandsLock.RUnlock()
}

// ExecuteCommand executes the command specified by message, if it exists.
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

	if len(args) == 2 {
		message = args[1]
	} else {
		message = ""
	}

	event := EventCommand{
		sender, command, message,
		sender.CanExecute(command),
	}
	server.FireEvent(EventTypeCommand, &event)
	if !event.Allow {
		sender.SendMessage("You do not have permission to execute this command!")
		return
	}

	go command.Handler(sender, command, message)
}

// AddHandler registers a handler for the specified event type.
func (server *Server) AddHandler(eventType EventType, handler EventHandler) {
	server.handlersLock.Lock()
	server.handlers[eventType] = append(server.handlers[eventType], handler)
	server.handlersLock.Unlock()
}

// FireEvent dispatches event to the server.
func (server *Server) FireEvent(eventType EventType, event interface{}) {
	server.handlersLock.RLock()
	for _, handler := range server.handlers[eventType] {
		handler(eventType, event)
	}
	server.handlersLock.RUnlock()
}

// AddGenerator registers a level generator.
func (server *Server) AddGenerator(name string, fn GeneratorFunc) {
	server.generatorsLock.Lock()
	server.generators[name] = fn
	server.generatorsLock.Unlock()
}

// NewGenerator returns a new generator with the specified options.
func (server *Server) NewGenerator(name string, args ...string) Generator {
	server.generatorsLock.RLock()
	defer server.generatorsLock.RUnlock()
	if fn := server.generators[name]; fn != nil {
		return fn(args...)
	}

	return nil
}

// RegisterPlugin registers and enables plugin.
func (server *Server) RegisterPlugin(plugin Plugin) {
	server.pluginsLock.Lock()
	server.plugins = append(server.plugins, plugin)
	server.pluginsLock.Unlock()

	plugin.Enable(server)
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

func (server *Server) run(wg *sync.WaitGroup) {
	wg.Add(1)

	server.updateTicker = time.NewTicker(UpdateInterval)
	go func() {
		for range server.updateTicker.C {
			server.ForEachEntity(func(entity *Entity) {
				entity.update()
			})

			server.ForEachLevel(func(level *Level) {
				level.update()
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
			server.updateTicker.Stop()
			server.saveTicker.Stop()
			if server.heartbeatTicker != nil {
				server.heartbeatTicker.Stop()
			}

			server.playersLock.RLock()
			players := make([]*Player, len(server.players))
			copy(players, server.players)
			server.playersLock.RUnlock()

			for _, player := range players {
				player.Kick("Server shutting down!")
			}

			server.levelsLock.Lock()
			for _, level := range server.levels {
				server.SaveLevel(level)
			}
			server.levels = nil
			server.levelsLock.Unlock()

			server.pluginsLock.Lock()
			for _, plugin := range server.plugins {
				plugin.Disable(server)
			}
			server.plugins = nil
			server.pluginsLock.Unlock()

			wg.Done()
			return

		default:
			conn, err := server.listener.Accept()
			if err != nil {
				continue
			}

			player := NewPlayer(conn, server)
			go player.handle()
		}
	}
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
		log.Printf("sendHeartbeat: %s\n", err.Error())
		return
	}

	if response.StatusCode != 200 {
		log.Printf("sendHeartbeat: %s\n", response.Status)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("sendHeartbeat: %s\n", err.Error())
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
		log.Printf("sendHeartbeat: %s\n", data.Errors[0][0])
		return
	}

	server.URL = string(body)
}
