// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/structinf/Go-MCC/gomcc"
)

const (
	PermOperator = 1 << 0
	PermBan      = 1 << 1
	PermKick     = 1 << 2
	PermChat     = 1 << 3
	PermTeleport = 1 << 4
	PermSummon   = 1 << 5
	PermLevel    = 1 << 6
)

type level struct {
	*gomcc.Level

	motd    string
	physics bool

	simulators []gomcc.Simulator
}

type player struct {
	*gomcc.Player

	mute       bool
	ignoreList []string
	firstLogin time.Time
	lastLogin  time.Time

	lastSender   string
	lastLevel    *gomcc.Level
	lastLocation gomcc.Location
}

func (player *player) isIgnored(name string) bool {
	for _, p := range player.ignoreList {
		if p == name {
			return true
		}
	}

	return false
}

type Plugin struct {
	db *db

	defaultRank string
	ranks       map[string]*gomcc.Rank
	ranksLock   sync.RWMutex

	levels     map[string]*level
	levelsLock sync.RWMutex

	players     map[string]*player
	playersLock sync.RWMutex
}

func Initialize() gomcc.Plugin {
	db := newDb("core.db")
	if db == nil {
		return nil
	}

	return &Plugin{
		db:      db,
		levels:  make(map[string]*level),
		players: make(map[string]*player),
	}
}

func (plugin *Plugin) Name() string {
	return "Core"
}

func (plugin *Plugin) Enable(server *gomcc.Server) {
	plugin.loadRanks()

	server.RegisterCommand(&gomcc.Command{
		Name:        "back",
		Description: "Return to your location before your last teleportation.",
		Usage:       "/back",
		Permissions: PermTeleport,
		Handler:     plugin.handleBack,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "ban",
		Description: "Ban a player from the server.",
		Usage:       "/ban <player> [reason]",
		Permissions: PermBan,
		Handler:     plugin.handleBan,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "banip",
		Description: "Ban an IP address from the server.",
		Usage:       "/banip <ip> [reason]",
		Permissions: PermBan,
		Handler:     plugin.handleBanIp,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "commands",
		Description: "List all commands.",
		Usage:       "/commands",
		Handler:     plugin.handleCommands,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "copylvl",
		Description: "Copy a level.",
		Usage:       "/copylvl <src> <dst>",
		Permissions: PermLevel,
		Handler:     plugin.handleCopyLvl,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "env",
		Description: "Change the environment of the current level.",
		Usage:       "/env <option> <value>\n/env reset",
		Permissions: PermLevel,
		Handler:     plugin.handleEnv,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "goto",
		Description: "Move to another level.",
		Usage:       "/goto <level>",
		Handler:     plugin.handleGoto,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "help",
		Description: "Describe a command.",
		Usage:       "/help <command>",
		Handler:     plugin.handleHelp,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "ignore",
		Description: "Ignore chat from a player",
		Usage:       "/ignore [player]",
		Handler:     plugin.handleIgnore,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "kick",
		Description: "Kick a player from the server.",
		Usage:       "/kick <player> [reason]",
		Permissions: PermKick,
		Handler:     plugin.handleKick,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "levels",
		Description: "List all loaded levels.",
		Usage:       "/levels",
		Handler:     plugin.handleLevels,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "load",
		Description: "Load a level.",
		Usage:       "/load <level>",
		Permissions: PermLevel,
		Handler:     plugin.handleLoad,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "main",
		Description: "Set the main level.",
		Usage:       "/main [level]",
		Permissions: PermLevel,
		Handler:     plugin.handleMain,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "me",
		Description: "Broadcast an action.",
		Usage:       "/me <action>",
		Handler:     plugin.handleMe,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "mute",
		Description: "Mute a player.",
		Usage:       "/mute <player>",
		Permissions: PermChat,
		Handler:     plugin.handleMute,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "newlvl",
		Description: "Create a new level.",
		Usage:       "/newlvl <name> <width> <height> <length> <theme> [<args>...]",
		Permissions: PermLevel,
		Handler:     plugin.handleNewLvl,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "nick",
		Description: "Set the nickname of a player",
		Usage:       "/nick <player> [nick]",
		Permissions: PermChat,
		Handler:     plugin.handleNick,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "players",
		Description: "List all players.",
		Usage:       "/players [level]",
		Handler:     plugin.handlePlayers,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "physics",
		Description: "Set the physics state of a level.",
		Usage:       "/physics <level> <value>\n/physics <value>",
		Permissions: PermLevel,
		Handler:     plugin.handlePhysics,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "r",
		Description: "Reply to the last message.",
		Usage:       "/r <message>",
		Handler:     plugin.handleR,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "rank",
		Description: "Set the rank of a player.",
		Usage:       "/rank <player> [rank]",
		Permissions: PermOperator,
		Handler:     plugin.handleRank,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "save",
		Description: "Save a level.",
		Usage:       "/save <level>\n/save all",
		Permissions: PermLevel,
		Handler:     plugin.handleSave,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "say",
		Description: "Broadcast a message.",
		Usage:       "/say <message>",
		Permissions: PermChat,
		Handler:     plugin.handleSay,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "seen",
		Description: "Check when a player was last online.",
		Usage:       "/seen <player>",
		Handler:     plugin.handleSeen,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "setspawn",
		Description: "Set the spawn location of the level to your location.",
		Usage:       "/setspawn [player]",
		Permissions: PermLevel,
		Handler:     plugin.handleSetSpawn,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "skin",
		Description: "Set the skin of a player.",
		Usage:       "/skin <player> <skin>",
		Permissions: PermOperator,
		Handler:     plugin.handleSkin,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "spawn",
		Description: "Teleport to the spawn location of the level.",
		Usage:       "/spawn",
		Handler:     plugin.handleSpawn,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "summon",
		Description: "Summon a player to your location.",
		Usage:       "/summon <player>\n/summon all",
		Permissions: PermSummon,
		Handler:     plugin.handleSummon,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "unload",
		Description: "Unload a level.",
		Usage:       "/unload <level>",
		Permissions: PermLevel,
		Handler:     plugin.handleUnload,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "tell",
		Description: "Send a private message to a player.",
		Usage:       "/tell <player> <message>",
		Handler:     plugin.handleTell,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "tp",
		Description: "Teleport to another player.",
		Usage:       "/tp <player>\n/tp <x> <y> <z>",
		Permissions: PermTeleport,
		Handler:     plugin.handleTp,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "unban",
		Description: "Remove the ban for a player.",
		Usage:       "/unban <player>",
		Permissions: PermBan,
		Handler:     plugin.handleUnban,
	})

	server.RegisterCommand(&gomcc.Command{
		Name:        "unbanip",
		Description: "Remove the ban for an IP address.",
		Usage:       "/unbanip <ip>",
		Permissions: PermBan,
		Handler:     plugin.handleUnbanIp,
	})

	server.RegisterHandler(gomcc.EventTypePlayerLogin, plugin.handlePlayerLogin)
	server.RegisterHandler(gomcc.EventTypePlayerChat, plugin.handlePlayerChat)

	server.RegisterHandler(gomcc.EventTypePlayerJoin, func(eventType gomcc.EventType, event interface{}) {
		e := event.(*gomcc.EventPlayerJoin)
		plugin.addPlayer(e.Player)
	})

	server.RegisterHandler(gomcc.EventTypePlayerQuit, func(eventType gomcc.EventType, event interface{}) {
		e := event.(*gomcc.EventPlayerQuit)
		player := plugin.findPlayer(e.Player.Name())
		plugin.savePlayer(player)
		plugin.removePlayer(e.Player)
	})

	server.RegisterHandler(gomcc.EventTypeLevelLoad, func(eventType gomcc.EventType, event interface{}) {
		e := event.(*gomcc.EventLevelLoad)
		plugin.addLevel(e.Level)
	})

	server.RegisterHandler(gomcc.EventTypeLevelUnload, func(eventType gomcc.EventType, event interface{}) {
		e := event.(*gomcc.EventLevelUnload)
		level := plugin.findLevel(e.Level.Name)
		plugin.saveLevel(level)
		plugin.removeLevel(e.Level)
	})

	server.ForEachPlayer(func(player *gomcc.Player) {
		plugin.addPlayer(player)
	})

	server.ForEachLevel(func(level *gomcc.Level) {
		plugin.addLevel(level)
	})
}

func (plugin *Plugin) Disable(server *gomcc.Server) {
	plugin.playersLock.Lock()
	for _, player := range plugin.players {
		plugin.savePlayer(player)
	}
	plugin.players = nil
	plugin.playersLock.Unlock()

	plugin.levelsLock.Lock()
	for _, level := range plugin.levels {
		plugin.saveLevel(level)
	}
	plugin.levels = nil
	plugin.levelsLock.Unlock()

	plugin.db.Close()
}

func (plugin *Plugin) loadRanks() {
	plugin.ranksLock.Lock()
	defer plugin.ranksLock.Unlock()

	plugin.ranks = make(map[string]*gomcc.Rank)
	for _, r := range plugin.db.queryRanks() {
		plugin.ranks[r.Name] = &gomcc.Rank{
			Name:        r.Name,
			Tag:         r.Tag.String,
			Permissions: r.Permissions,
			CanPlace:    gomcc.DefaultRank.CanPlace,
			CanBreak:    gomcc.DefaultRank.CanBreak,
		}
	}

	for _, rule := range plugin.db.queryCommandRules() {
		if rank := plugin.ranks[rule.Rank]; rank != nil {
			if rank.Rules == nil {
				rank.Rules = make(map[string]bool)
			}

			rank.Rules[rule.Command] = rule.Access
		}
	}

	for _, rule := range plugin.db.queryBlockRules() {
		rank := plugin.ranks[rule.Rank]
		if rank != nil && rule.BlockID >= 0 && rule.BlockID <= gomcc.BlockMax {
			switch rule.Action {
			case 0:
				rank.CanBreak[rule.BlockID] = rule.Access
			case 1:
				rank.CanPlace[rule.BlockID] = rule.Access
			}
		}
	}

	plugin.defaultRank = plugin.db.queryConfig("default_rank")
}

func (plugin *Plugin) findRank(name string) *gomcc.Rank {
	plugin.ranksLock.RLock()
	defer plugin.ranksLock.RUnlock()
	return plugin.ranks[name]
}

func (plugin *Plugin) addPlayer(p *gomcc.Player) *player {
	name := p.Name()

	db, ok := plugin.db.queryPlayer(name)
	if !ok {
		db.Rank = sql.NullString{plugin.defaultRank, true}
		db.FirstLogin = time.Now()
		db.Nickname = name
	}

	player := &player{
		Player:     p,
		firstLogin: db.FirstLogin,
		lastLogin:  time.Now(),
	}

	player.Nickname = db.Nickname
	if len(db.IgnoreList) != 0 {
		player.ignoreList = strings.Split(db.IgnoreList, ",")
	}
	if db.Rank.Valid {
		player.Rank = plugin.findRank(db.Rank.String)
	}

	plugin.playersLock.Lock()
	plugin.players[name] = player
	plugin.playersLock.Unlock()
	return player
}

func (plugin *Plugin) removePlayer(player *gomcc.Player) {
	plugin.playersLock.Lock()
	delete(plugin.players, player.Name())
	plugin.playersLock.Unlock()
}

func (plugin *Plugin) findPlayer(name string) *player {
	plugin.playersLock.RLock()
	defer plugin.playersLock.RUnlock()
	return plugin.players[name]
}

func (plugin *Plugin) savePlayer(player *player) {
	var rank sql.NullString
	if player.Rank != nil {
		rank = sql.NullString{player.Rank.Name, true}
	}

	plugin.db.updatePlayer(player.Name(), &dbPlayer{
		Rank:       rank,
		FirstLogin: player.firstLogin,
		LastLogin:  player.lastLogin,
		Nickname:   player.Nickname,
		IgnoreList: strings.Join(player.ignoreList, ","),
		Mute:       player.mute,
	})
}

func (plugin *Plugin) addLevel(l *gomcc.Level) *level {
	name := l.Name

	db, _ := plugin.db.queryLevel(name)
	level := &level{
		Level:   l,
		motd:    db.MOTD,
		physics: db.Physics,
	}

	parseMOTD(db.MOTD, &level.HackConfig)

	plugin.disablePhysics(level)
	if db.Physics {
		plugin.enablePhysics(level)
	}

	plugin.levelsLock.Lock()
	plugin.levels[name] = level
	plugin.levelsLock.Unlock()
	return level
}

func (plugin *Plugin) removeLevel(level *gomcc.Level) {
	plugin.levelsLock.Lock()
	delete(plugin.levels, level.Name)
	plugin.levelsLock.Unlock()
}

func (plugin *Plugin) findLevel(name string) *level {
	plugin.levelsLock.RLock()
	defer plugin.levelsLock.RUnlock()
	return plugin.levels[name]
}

func (plugin *Plugin) saveLevel(level *level) {
	plugin.db.updateLevel(level.Name, &dbLevel{
		MOTD:    level.motd,
		Physics: level.physics,
	})
}

func (plugin *Plugin) handlePlayerLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerLogin)
	addr := e.Player.RemoteAddr()
	name := e.Player.Name()
	e.Cancel, e.CancelReason = plugin.db.checkBan(addr, name)
}

func (plugin *Plugin) handlePlayerChat(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerChat)
	name := e.Player.Name()
	player := plugin.findPlayer(name)
	if player.mute {
		player.SendMessage("You are muted")
		e.Cancel = true
		return
	}

	for i := len(e.Targets) - 1; i >= 0; i-- {
		if plugin.findPlayer(e.Targets[i].Name()).isIgnored(name) {
			e.Targets = append(e.Targets[:i], e.Targets[i+1:]...)
		}
	}
}
