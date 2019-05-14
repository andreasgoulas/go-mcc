// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"sync"

	"github.com/structinf/Go-MCC/gomcc"
)

type Rank struct {
	Permissions []string `json:"permissions,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Suffix      string   `json:"suffix,omitempty"`
}

type RankManager struct {
	Lock    sync.RWMutex     `json:"-"`
	Ranks   map[string]*Rank `json:"ranks"`
	Default string           `json:"default"`
}

func (manager *RankManager) Load(path string) {
	manager.Lock.Lock()
	loadJson(path, manager)
	manager.Lock.Unlock()
}

func (manager *RankManager) Save(path string) {
	manager.Lock.RLock()
	saveJson(path, manager)
	manager.Lock.RUnlock()
}

func (manager *RankManager) Rank(name string) *Rank {
	manager.Lock.Lock()
	defer manager.Lock.Unlock()
	return manager.Ranks[name]
}

func (manager *RankManager) Update(player *Player) {
	if player.PermGroup == nil {
		player.PermGroup = &gomcc.PermissionGroup{}
		player.Player.AddPermissionGroup(player.PermGroup)
	}

	player.PermGroup.Clear()
	for _, perm := range player.Permissions {
		player.PermGroup.AddPermission(perm)
	}

	if rank := manager.Rank(player.Rank); rank != nil {
		for _, perm := range rank.Permissions {
			player.PermGroup.AddPermission(perm)
		}
	}
}
