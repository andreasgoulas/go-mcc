// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"sync"
	"time"

	"github.com/structinf/Go-MCC/gomcc"
)

type OfflinePlayer struct {
	Rank        string    `json:"rank"`
	FirstLogin  time.Time `json:"first-login"`
	LastLogin   time.Time `json:"last-login"`
	Nickname    string    `json:"nickname,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	Ignore      []string  `json:"ignore,omitempty"`
	Mute        bool      `json:"mute"`
}

func (player *OfflinePlayer) IsIgnored(name string) bool {
	for _, p := range player.Ignore {
		if p == name {
			return true
		}
	}

	return false
}

type Player struct {
	*OfflinePlayer

	Player    *gomcc.Player
	PermGroup *gomcc.PermissionGroup

	LastSender   string
	LastLevel    *gomcc.Level
	LastLocation gomcc.Location
}

type PlayerManager struct {
	Lock           sync.RWMutex
	Players        map[string]*Player
	OfflinePlayers map[string]*OfflinePlayer
}

func (manager *PlayerManager) Load(path string) {
	manager.Lock.Lock()
	manager.Players = make(map[string]*Player)
	loadJson(path, &manager.OfflinePlayers)
	manager.Lock.Unlock()
}

func (manager *PlayerManager) Save(path string) {
	manager.Lock.RLock()
	saveJson(path, &manager.OfflinePlayers)
	manager.Lock.RUnlock()
}

func (manager *PlayerManager) OfflinePlayer(name string) *OfflinePlayer {
	manager.Lock.RLock()
	defer manager.Lock.RUnlock()
	return manager.OfflinePlayers[name]
}

func (manager *PlayerManager) Player(name string) *Player {
	manager.Lock.RLock()
	defer manager.Lock.RUnlock()
	return manager.Players[name]
}

func (manager *PlayerManager) Add(player *gomcc.Player) (cplayer *Player, ok bool) {
	name := player.Name()

	manager.Lock.RLock()
	data, ok := manager.OfflinePlayers[name]
	manager.Lock.RUnlock()

	if !ok {
		manager.Lock.Lock()
		data = &OfflinePlayer{}
		manager.OfflinePlayers[name] = data
		manager.Lock.Unlock()
	}

	manager.Lock.Lock()
	cplayer = &Player{OfflinePlayer: data, Player: player}
	manager.Players[name] = cplayer
	manager.Lock.Unlock()

	return
}

func (manager *PlayerManager) Remove(player *gomcc.Player) {
	manager.Lock.Lock()
	delete(manager.Players, player.Name())
	manager.Lock.Unlock()
}
