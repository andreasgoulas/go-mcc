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
	CoreRanks.Load("ranks.json")
	CorePlayers.Load("players.json")

	server.RegisterCommand(&commandBack)
	server.RegisterCommand(&commandBan)
	server.RegisterCommand(&commandBanIp)
	server.RegisterCommand(&commandCommands)
	server.RegisterCommand(&commandCopyLvl)
	server.RegisterCommand(&commandGoto)
	server.RegisterCommand(&commandHelp)
	server.RegisterCommand(&commandIgnore)
	server.RegisterCommand(&commandKick)
	server.RegisterCommand(&commandLevels)
	server.RegisterCommand(&commandLoad)
	server.RegisterCommand(&commandMain)
	server.RegisterCommand(&commandMe)
	server.RegisterCommand(&commandMute)
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
	CorePlayers.Save("players.json")
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
	CorePlayers.OnJoin(e.Player, CoreRanks.Default)
}

func handlePlayerQuit(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerQuit)
	CorePlayers.OnQuit(e.Player)
}

func handlePlayerChat(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerChat)
	name := e.Player.Name()

	player := CorePlayers.Player(name)
	if player.Mute {
		e.Player.SendMessage("You are muted")
		e.Cancel = true
		return
	}

	if rank := CoreRanks.Rank(player.Rank); rank != nil {
		e.Format = fmt.Sprintf("%s%%s%s: &f%%s", rank.Prefix, rank.Suffix)
	}

	for i := len(e.Targets) - 1; i >= 0; i-- {
		if CorePlayers.Player(e.Targets[i].Name()).IsIgnored(name) {
			e.Targets = append(e.Targets[:i], e.Targets[i+1:]...)
		}
	}
}
