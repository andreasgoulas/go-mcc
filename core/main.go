// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/structinf/Go-MCC/gomcc"
)

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

type level struct {
	*gomcc.Level

	simulators []gomcc.Simulator
}

type player struct {
	*gomcc.Player

	permGroup *gomcc.PermissionGroup

	mute       bool
	ignoreList []string
	msgFormat  string

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
	db *sqlx.DB

	levels     map[string]*level
	levelsLock sync.RWMutex

	players     map[string]*player
	playersLock sync.RWMutex
}

func Initialize() gomcc.Plugin {
	db, err := sqlx.Open("sqlite3", "core.db")
	if err != nil {
		log.Println(err)
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
	plugin.db.MustExec(`
CREATE TABLE IF NOT EXISTS banned_names(
	name TEXT PRIMARY KEY,
	reason TEXT,
	banned_by TEXT,
	timestamp DATETIME
);

CREATE TABLE IF NOT EXISTS banned_ips(
	ip TEXT PRIMARY KEY,
	reason TEXT,
	banned_by TEXT,
	timestamp DATETIME
);

CREATE TABLE IF NOT EXISTS levels(
	name TEXT PRIMARY KEY,
	motd TEXT,
	physics INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS ranks(
	name TEXT PRIMARY KEY,
	permissions TEXT,
	prefix TEXT,
	suffix TEXT,
	is_default INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS players(
	name TEXT PRIMARY KEY,
	rank TEXT,
	first_login DATETIME,
	last_login DATETIME,
	permissions TEXT,
	nickname TEXT,
	ignore_list TEXT,
	mute INTEGER NOT NULL,
	FOREIGN KEY(rank) REFERENCES ranks(name)
);`)

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
		Name:        "physics",
		Description: "Set the physics state of a level.",
		Permission:  "core.physics",
		Handler:     plugin.handlePhysics,
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
	server.RegisterHandler(gomcc.EventTypeLevelLoad, plugin.handleLevelLoad)
	server.RegisterHandler(gomcc.EventTypeLevelUnload, plugin.handleLevelUnload)

	server.ForEachLevel(func(level *gomcc.Level) {
		plugin.handleLevelLoad(gomcc.EventTypeLevelLoad, &gomcc.EventLevelLoad{level})
	})
}

func (plugin *Plugin) Disable(server *gomcc.Server) {
	plugin.db.Close()
}

func (plugin *Plugin) addPlayer(ptr *gomcc.Player) *player {
	name := ptr.Name()
	player := &player{Player: ptr}

	plugin.db.MustExec(`INSERT OR IGNORE INTO players(name, rank, first_login, mute)
VALUES(?, (SELECT name FROM ranks WHERE is_default = 1), CURRENT_TIMESTAMP, 0);`, name)
	plugin.db.MustExec("UPDATE players SET last_login = CURRENT_TIMESTAMP WHERE name = ?;", name)

	data := struct {
		Nickname   sql.NullString `db:"nickname"`
		IgnoreList sql.NullString `db:"ignore_list"`
		Mute       bool           `db:"mute"`
	}{}
	plugin.db.Get(&data, "SELECT nickname, ignore_list, mute FROM players WHERE name = ?", name)

	player.mute = data.Mute
	if data.Nickname.Valid {
		player.Nickname = data.Nickname.String
	}
	if data.IgnoreList.Valid && len(data.IgnoreList.String) != 0 {
		player.ignoreList = strings.Split(data.IgnoreList.String, ",")
	}

	plugin.updatePermissions(player)

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

func (plugin *Plugin) updatePermissions(player *player) {
	if player.permGroup == nil {
		player.permGroup = &gomcc.PermissionGroup{}
		player.AddPermissionGroup(player.permGroup)
	}

	player.permGroup.Clear()

	playerData := struct {
		Rank        sql.NullString `db:"rank"`
		Permissions sql.NullString `db:"permissions"`
	}{}
	plugin.db.Get(&playerData, "SELECT rank, permissions FROM players WHERE name = ?", player.Name())
	if playerData.Permissions.Valid {
		for _, perm := range strings.Split(playerData.Permissions.String, ",") {
			player.permGroup.AddPermission(perm)
		}
	}

	rankData := struct {
		Prefix      sql.NullString `db:"prefix"`
		Suffix      sql.NullString `db:"suffix"`
		Permissions sql.NullString `db:"permissions"`
	}{}
	if playerData.Rank.Valid {
		plugin.db.Get(&rankData, "SELECT prefix, suffix, permissions FROM ranks WHERE name = ?", playerData.Rank.String)
		if rankData.Permissions.Valid {
			for _, perm := range strings.Split(rankData.Permissions.String, ",") {
				player.permGroup.AddPermission(perm)
			}
		}
	}

	player.msgFormat = fmt.Sprintf("%s%%s%s: &f%%s", rankData.Prefix.String, rankData.Suffix.String)
}

func (plugin *Plugin) addLevel(ptr *gomcc.Level) *level {
	name := ptr.Name
	level := &level{Level: ptr}

	plugin.db.MustExec("INSERT OR IGNORE INTO levels(name, physics) VALUES(?, 0)", name)

	data := struct {
		MOTD    sql.NullString `db:"motd"`
		Physics bool           `db:"physics"`
	}{}
	plugin.db.Get(&data, "SELECT motd, physics FROM levels WHERE name = ?", name)

	if data.MOTD.Valid {
		level.MOTD = data.MOTD.String
	}

	plugin.disablePhysics(level)
	if data.Physics {
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

func (plugin *Plugin) handlePlayerLogin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerLogin)

	var reason sql.NullString
	addr := e.Player.RemoteAddr()
	name := e.Player.Name()
	if plugin.db.Get(&reason, `SELECT reason FROM banned_ips WHERE ip = ? UNION
SELECT reason FROM banned_names WHERE name = ?`, addr, name) == nil {
		e.Cancel = true
		e.CancelReason = reason.String
		return
	}
}

func (plugin *Plugin) handlePlayerJoin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerJoin)
	plugin.addPlayer(e.Player)
}

func (plugin *Plugin) handlePlayerQuit(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerQuit)
	plugin.removePlayer(e.Player)
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

	e.Format = player.msgFormat
	for i := len(e.Targets) - 1; i >= 0; i-- {
		if plugin.findPlayer(e.Targets[i].Name()).isIgnored(name) {
			e.Targets = append(e.Targets[:i], e.Targets[i+1:]...)
		}
	}
}

func (plugin *Plugin) handleLevelLoad(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventLevelLoad)
	plugin.addLevel(e.Level)
}

func (plugin *Plugin) handleLevelUnload(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventLevelUnload)
	plugin.removeLevel(e.Level)
}
