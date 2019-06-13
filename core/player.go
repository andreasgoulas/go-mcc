// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"sync"
	"time"

	"github.com/structinf/Go-MCC/gomcc"
)

type PlayerInfo struct {
	Rank        string    `json:"rank"`
	FirstLogin  time.Time `json:"first-login"`
	LastLogin   time.Time `json:"last-login"`
	Nickname    string    `json:"nickname,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	Ignore      []string  `json:"ignore,omitempty"`
	Mute        bool      `json:"mute"`

	Player *Player `json:"-"`
}

func (player *PlayerInfo) IsIgnored(name string) bool {
	for _, p := range player.Ignore {
		if p == name {
			return true
		}
	}

	return false
}

type Player struct {
	*gomcc.Player
	PermGroup *gomcc.PermissionGroup

	LastSender   string
	LastLevel    *gomcc.Level
	LastLocation gomcc.Location
}

type PlayerManager struct {
	Lock    sync.RWMutex
	Players map[string]*PlayerInfo
}

func (manager *PlayerManager) Load(path string) {
	manager.Lock.Lock()
	manager.Players = make(map[string]*PlayerInfo)
	loadJson(path, &manager.Players)
	manager.Lock.Unlock()
}

func (manager *PlayerManager) Save(path string) {
	manager.Lock.RLock()
	saveJson(path, &manager.Players)
	manager.Lock.RUnlock()
}

func (manager *PlayerManager) Find(name string) *PlayerInfo {
	manager.Lock.RLock()
	defer manager.Lock.RUnlock()
	return manager.Players[name]
}

func (manager *PlayerManager) Add(player *gomcc.Player) (info *PlayerInfo, firstLogin bool) {
	name := player.Name()
	manager.Lock.RLock()
	defer manager.Lock.RUnlock()

	info, ok := manager.Players[name]
	if !ok {
		info = &PlayerInfo{}
		manager.Players[name] = info
	}

	info.Player = &Player{Player: player}
	return info, !ok
}

func (manager *PlayerManager) Remove(player *gomcc.Player) {
	name := player.Name()
	manager.Lock.Lock()
	defer manager.Lock.Unlock()

	if info, ok := manager.Players[name]; ok {
		info.Player = nil
	}
}
