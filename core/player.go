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

package core

import (
	"sync"
	"time"

	"Go-MCC/gomcc"
)

type OfflinePlayer struct {
	Rank        string    `json:"rank"`
	FirstLogin  time.Time `json:"first-login"`
	LastLogin   time.Time `json:"last-login"`
	Nickname    string    `json:"nickname,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
}

type Player struct {
	*OfflinePlayer

	Player    *gomcc.Player
	PermGroup *gomcc.PermissionGroup

	LastSender   string
	LastLevel    *gomcc.Level
	LastLocation gomcc.Location
}

func (player *Player) UpdatePermissions() {
	if player.PermGroup == nil {
		player.PermGroup = &gomcc.PermissionGroup{}
		player.Player.AddPermissionGroup(player.PermGroup)
	}

	player.PermGroup.Clear()
	for _, perm := range player.Permissions {
		player.PermGroup.AddPermission(perm)
	}

	CoreRanks.Lock.RLock()
	defer CoreRanks.Lock.RUnlock()
	if rank, ok := CoreRanks.Ranks[player.Rank]; ok {
		for _, perm := range rank.Permissions {
			player.PermGroup.AddPermission(perm)
		}
	}
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

func (manager *PlayerManager) OnJoin(player *gomcc.Player, defaultRank string) {
	name := player.Name()

	manager.Lock.RLock()
	data, ok := manager.OfflinePlayers[name]
	manager.Lock.RUnlock()

	if !ok {
		manager.Lock.Lock()
		data = &OfflinePlayer{
			Rank:       defaultRank,
			Nickname:   "",
			FirstLogin: time.Now(),
		}
		manager.OfflinePlayers[name] = data
		manager.Lock.Unlock()
	}

	data.LastLogin = time.Now()
	if len(data.Nickname) != 0 {
		player.Nickname = data.Nickname
	}

	cplayer := &Player{OfflinePlayer: data, Player: player}
	cplayer.UpdatePermissions()

	manager.Lock.Lock()
	manager.Players[name] = cplayer
	manager.Lock.Unlock()
}

func (manager *PlayerManager) OnQuit(player *gomcc.Player) {
	manager.Lock.Lock()
	delete(manager.Players, player.Name())
	manager.Lock.Unlock()
}
