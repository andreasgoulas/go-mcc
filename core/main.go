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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"Go-MCC/gomcc"
)

type Rank struct {
	Permissions []string `json:"permissions"`
	Prefix      string   `json:"prefix"`
	Suffix      string   `json:"suffix"`
}

type RankManager struct {
	Lock    sync.RWMutex    `json:"-"`
	Ranks   map[string]Rank `json:"ranks"`
	Default string          `json:"default"`
}

type OfflinePlayer struct {
	Rank        string    `json:"rank"`
	Nickname    string    `json:"nickname"`
	FirstLogin  time.Time `json:"first-login"`
	LastLogin   time.Time `json:"last-login"`
	Permissions []string  `json:"permissions"`
}

type Player struct {
	*OfflinePlayer

	PermGroup *gomcc.PermissionGroup

	LastSender   string
	LastLevel    *gomcc.Level
	LastLocation gomcc.Location
}

func (player *Player) UpdatePermissions(p *gomcc.Player) {
	if player.PermGroup == nil {
		player.PermGroup = &gomcc.PermissionGroup{}
		p.AddPermissionGroup(player.PermGroup)
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
	Lock    sync.RWMutex
	Players map[string]*OfflinePlayer
	Online  map[string]*Player
}

var (
	CoreRanks   RankManager
	CoreBans    BanManager
	CorePlayers PlayerManager
)

func loadJson(path string, v interface{}) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	if err := json.Unmarshal(file, v); err != nil {
		log.Printf("loadJson: %s\n", err)
	}
}

func saveJson(path string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		log.Printf("saveJson: %s\n", err)
		return
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		log.Printf("saveJson: %s\n", err)
	}
}

func Enable(server *gomcc.Server) {
	CoreBans.Load("bans.json")

	CoreRanks.Lock.Lock()
	loadJson("ranks.json", &CoreRanks)
	CoreRanks.Lock.Unlock()

	CorePlayers.Lock.Lock()
	CorePlayers.Players = make(map[string]*OfflinePlayer)
	CorePlayers.Online = make(map[string]*Player)
	loadJson("players.json", &CorePlayers.Players)
	CorePlayers.Lock.Unlock()

	server.RegisterCommand(&commandBack)
	server.RegisterCommand(&commandBan)
	server.RegisterCommand(&commandBanIp)
	server.RegisterCommand(&commandCommands)
	server.RegisterCommand(&commandCopyLvl)
	server.RegisterCommand(&commandGoto)
	server.RegisterCommand(&commandHelp)
	server.RegisterCommand(&commandKick)
	server.RegisterCommand(&commandLevels)
	server.RegisterCommand(&commandLoad)
	server.RegisterCommand(&commandMain)
	server.RegisterCommand(&commandMe)
	server.RegisterCommand(&commandNewLvl)
	server.RegisterCommand(&commandNick)
	server.RegisterCommand(&commandPlayers)
	server.RegisterCommand(&commandR)
	server.RegisterCommand(&commandRank)
	server.RegisterCommand(&commandSave)
	server.RegisterCommand(&commandSay)
	server.RegisterCommand(&commandSeen)
	server.RegisterCommand(&commandSetSpawn)
	server.RegisterCommand(&commandSkin)
	server.RegisterCommand(&commandSpawn)
	server.RegisterCommand(&commandSummon)
	server.RegisterCommand(&commandTell)
	server.RegisterCommand(&commandTp)
	server.RegisterCommand(&commandUnban)
	server.RegisterCommand(&commandUnbanIp)
	server.RegisterCommand(&commandUnload)

	server.RegisterHandler(gomcc.EventTypePlayerPreLogin, handlePlayerPreLogin)
	server.RegisterHandler(gomcc.EventTypePlayerLogin, handlePlayerLogin)
	server.RegisterHandler(gomcc.EventTypePlayerJoin, handlePlayerJoin)
	server.RegisterHandler(gomcc.EventTypePlayerQuit, handlePlayerQuit)
	server.RegisterHandler(gomcc.EventTypePlayerChat, handlePlayerChat)
}

func Disable(server *gomcc.Server) {
	CoreBans.Save("bans.json")

	CorePlayers.Lock.Lock()
	saveJson("players.json", &CorePlayers.Players)
	CorePlayers.Lock.Unlock()
}

func handlePlayerPreLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerPreLogin)
	addr := e.Player.RemoteAddr()
	if entry := CoreBans.IP.IsBanned(addr); entry != nil {
		e.Cancel = true
		e.CancelReason = entry.Reason
		return
	}
}

func handlePlayerLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerLogin)
	name := e.Player.Name()
	if entry := CoreBans.Name.IsBanned(name); entry != nil {
		e.Cancel = true
		e.CancelReason = entry.Reason
		return
	}
}

func handlePlayerJoin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerJoin)
	name := e.Player.Name()

	CorePlayers.Lock.RLock()
	data, ok := CorePlayers.Players[name]
	CorePlayers.Lock.RUnlock()

	if !ok {
		CorePlayers.Lock.Lock()
		data = &OfflinePlayer{
			Rank:       CoreRanks.Default,
			Nickname:   "",
			FirstLogin: time.Now(),
		}
		CorePlayers.Players[name] = data
		CorePlayers.Lock.Unlock()
	}

	data.LastLogin = time.Now()
	if len(data.Nickname) != 0 {
		e.Player.Nickname = data.Nickname
	}

	player := &Player{OfflinePlayer: data}
	player.UpdatePermissions(e.Player)
	CorePlayers.Online[name] = player
}

func handlePlayerQuit(eventType gomcc.EventType, event interface{}) {
	CorePlayers.Lock.Lock()
	defer CorePlayers.Lock.Unlock()

	e := event.(*gomcc.EventPlayerQuit)
	delete(CorePlayers.Online, e.Player.Name())
}

func handlePlayerChat(eventType gomcc.EventType, event interface{}) {
	CorePlayers.Lock.RLock()
	defer CorePlayers.Lock.RUnlock()

	CoreRanks.Lock.RLock()
	defer CoreRanks.Lock.RUnlock()

	e := event.(*gomcc.EventPlayerChat)
	name := e.Player.Name()
	data := CorePlayers.Players[name]
	if rank, ok := CoreRanks.Ranks[data.Rank]; ok {
		e.Format = fmt.Sprintf("%s%%s%s: &f%%s", rank.Prefix, rank.Suffix)
	}
}
