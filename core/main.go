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

	"Go-MCC/gomcc"
)

type Rank struct {
	Permissions []string `json:"permissions"`
	Prefix      string   `json:"prefix"`
	Suffix      string   `json:"suffix"`
}

type RankConfig struct {
	Ranks   map[string]Rank `json:"ranks"`
	Default string          `json:"default"`
}

type playerData struct {
	LastSender string

	LastLevel    *gomcc.Level
	LastLocation gomcc.Location
}

var (
	CoreDb      *Database
	CoreRanks   RankConfig
	CorePlayers map[string]*playerData
)

func loadRanks(path string) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(file, &CoreRanks)
	if err != nil {
		log.Printf("Rank Config Error: %s", err)
	}

	return
}

func PlayerData(name string) *playerData {
	data, ok := CorePlayers[name]
	if !ok {
		data = &playerData{}
		CorePlayers[name] = data
	}

	return data
}

func Initialize(server *gomcc.Server) {
	loadRanks("ranks.json")
	CoreDb = newDatabase("core.sqlite")
	CorePlayers = make(map[string]*playerData)

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
	server.RegisterHandler(gomcc.EventTypePlayerChat, handlePlayerChat)
}

func handlePlayerPreLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerPreLogin)
	result, reason := CoreDb.IsBanned(BanTypeIp, e.Player.RemoteAddr())
	if result {
		e.Cancel = true
		e.CancelReason = reason
	}
}

func handlePlayerLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerLogin)
	result, reason := CoreDb.IsBanned(BanTypeName, e.Player.Name())
	if result {
		e.Cancel = true
		e.CancelReason = reason
	}
}

func handlePlayerJoin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerJoin)
	name := e.Player.Name()
	CoreDb.onLogin(name)

	rank, ok := CoreRanks.Ranks[CoreDb.Rank(name)]
	if ok {
		e.Player.SetPermissions(rank.Permissions)
	}
}

func handlePlayerChat(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerChat)
	rank, ok := CoreRanks.Ranks[CoreDb.Rank(e.Player.Name())]
	if ok {
		e.Format = fmt.Sprintf("%s%%s%s: &f%%s", rank.Prefix, rank.Suffix)
	}
}
