// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/structinf/Go-MCC/gomcc"
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

type CorePlugin struct {
	Ranks   RankManager
	Bans    BanManager
	Players PlayerManager
}

func Initialize() gomcc.Plugin {
	return &CorePlugin{}
}

func (plugin *CorePlugin) Name() string {
	return "Core"
}

func (plugin *CorePlugin) Enable(server *gomcc.Server) {
	plugin.Bans.Load("bans.json")
	plugin.Ranks.Load("ranks.json")
	plugin.Players.Load("players.json")

	server.RegisterCommand(&gomcc.Command{
		Name:        "back",
		Description: "Return to your location before your last teleportation.",
		Permission:  "core.back",
		Handler:     plugin.handleBack,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "ban",
		Description: "Ban a player from the server.",
		Permission:  "core.ban",
		Handler:     plugin.handleBan,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "banip",
		Description: "Ban an IP address from the server.",
		Permission:  "core.banip",
		Handler:     plugin.handleBanIp,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "commands",
		Description: "List all commands.",
		Permission:  "core.commands",
		Handler:     plugin.handleCommands,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "copylvl",
		Description: "Copy a level.",
		Permission:  "core.copylvl",
		Handler:     plugin.handleCopyLvl,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "env",
		Description: "Change the environment of the current level.",
		Permission:  "core.env",
		Handler:     plugin.handleEnv,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "goto",
		Description: "Move to another level.",
		Permission:  "core.goto",
		Handler:     plugin.handleGoto,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "help",
		Description: "Describe a command.",
		Permission:  "core.help",
		Handler:     plugin.handleHelp,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "ignore",
		Description: "Ignore chat from a player",
		Permission:  "core.ignore",
		Handler:     plugin.handleIgnore,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "kick",
		Description: "Kick a player from the server.",
		Permission:  "core.kick",
		Handler:     plugin.handleKick,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "levels",
		Description: "List all loaded levels.",
		Permission:  "core.levels",
		Handler:     plugin.handleLevels,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "load",
		Description: "Load a level.",
		Permission:  "core.load",
		Handler:     plugin.handleLoad,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "main",
		Description: "Set the main level.",
		Permission:  "core.main",
		Handler:     plugin.handleMain,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "me",
		Description: "Broadcast an action.",
		Permission:  "core.me",
		Handler:     plugin.handleMe,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "mute",
		Description: "Mute a player.",
		Permission:  "core.mute",
		Handler:     plugin.handleMute,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "newlvl",
		Description: "Create a new level.",
		Permission:  "core.newlvl",
		Handler:     plugin.handleNewLvl,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "nick",
		Description: "Set the nickname of a player",
		Permission:  "core.nick",
		Handler:     plugin.handleNick,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "players",
		Description: "List all players.",
		Permission:  "core.players",
		Handler:     plugin.handlePlayers,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "r",
		Description: "Reply to the last message.",
		Permission:  "core.r",
		Handler:     plugin.handleR,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "rank",
		Description: "Set the rank of a player.",
		Permission:  "core.rank",
		Handler:     plugin.handleRank,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "save",
		Description: "Save a level.",
		Permission:  "core.save",
		Handler:     plugin.handleSave,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "say",
		Description: "Broadcast a message.",
		Permission:  "core.say",
		Handler:     plugin.handleSay,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "seen",
		Description: "Check when a player was last online.",
		Permission:  "core.seen",
		Handler:     plugin.handleSeen,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "setspawn",
		Description: "Set the spawn location of the level to your location.",
		Permission:  "core.setspawn",
		Handler:     plugin.handleSetSpawn,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "skin",
		Description: "Set the skin of a player.",
		Permission:  "core.skin",
		Handler:     plugin.handleSkin,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "spawn",
		Description: "Teleport to the spawn location of the level.",
		Permission:  "core.spawn",
		Handler:     plugin.handleSpawn,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "summon",
		Description: "Summon a player to your location.",
		Permission:  "core.summon",
		Handler:     plugin.handleSummon,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "unload",
		Description: "Unload a level.",
		Permission:  "core.unload",
		Handler:     plugin.handleUnload,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "tell",
		Description: "Send a private message to a player.",
		Permission:  "core.tell",
		Handler:     plugin.handleTell,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "tp",
		Description: "Teleport to another player.",
		Permission:  "core.tp",
		Handler:     plugin.handleTp,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "unban",
		Description: "Remove the ban for a player.",
		Permission:  "core.unban",
		Handler:     plugin.handleUnban,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "unbanip",
		Description: "Remove the ban for an IP address.",
		Permission:  "core.unbanip",
		Handler:     plugin.handleUnbanIp,
	})

	server.RegisterHandler(gomcc.EventTypePlayerLogin, plugin.handlePlayerLogin)
	server.RegisterHandler(gomcc.EventTypePlayerJoin, plugin.handlePlayerJoin)
	server.RegisterHandler(gomcc.EventTypePlayerQuit, plugin.handlePlayerQuit)
	server.RegisterHandler(gomcc.EventTypePlayerChat, plugin.handlePlayerChat)

	server.RegisterSimulator(plugin.handlePhysics)
}

func (plugin *CorePlugin) Disable(server *gomcc.Server) {
	plugin.Bans.Save("bans.json")
	plugin.Players.Save("players.json")
}

func (plugin *CorePlugin) handlePlayerLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerLogin)
	addr := e.Player.RemoteAddr()
	if entry := plugin.Bans.IP.IsBanned(addr); entry != nil {
		e.Cancel = true
		e.CancelReason = entry.Reason
		return
	}

	name := e.Player.Name()
	if entry := plugin.Bans.Name.IsBanned(name); entry != nil {
		e.Cancel = true
		e.CancelReason = entry.Reason
		return
	}
}

func (plugin *CorePlugin) handlePlayerJoin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerJoin)
	info, firstLogin := plugin.Players.Add(e.Player)
	if firstLogin {
		info.FirstLogin = time.Now()
		info.Rank = plugin.Ranks.Default
	}

	info.LastLogin = time.Now()
	if len(info.Nickname) != 0 {
		e.Player.Nickname = info.Nickname
	}

	plugin.Ranks.SetPermissions(info)
}

func (plugin *CorePlugin) handlePlayerQuit(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerQuit)
	plugin.Players.Remove(e.Player)
}

func (plugin *CorePlugin) handlePlayerChat(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerChat)
	name := e.Player.Name()

	info := plugin.Players.Find(name)
	if info.Mute {
		e.Player.SendMessage("You are muted")
		e.Cancel = true
		return
	}

	if rank := plugin.Ranks.Find(info.Rank); rank != nil {
		e.Format = fmt.Sprintf("%s%%s%s: &f%%s", rank.Prefix, rank.Suffix)
	}

	for i := len(e.Targets) - 1; i >= 0; i-- {
		if plugin.Players.Find(e.Targets[i].Name()).IsIgnored(name) {
			e.Targets = append(e.Targets[:i], e.Targets[i+1:]...)
		}
	}
}
