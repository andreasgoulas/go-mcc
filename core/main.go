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

type Level struct {
	*gomcc.Level

	Simulators []gomcc.Simulator
}

type Player struct {
	*gomcc.Player

	PermGroup *gomcc.PermissionGroup

	Mute       bool
	IgnoreList []string
	MsgFormat  string

	LastSender   string
	LastLevel    *gomcc.Level
	LastLocation gomcc.Location
}

func (player *Player) IsIgnored(name string) bool {
	for _, p := range player.IgnoreList {
		if p == name {
			return true
		}
	}

	return false
}

type CorePlugin struct {
	db *sqlx.DB

	levels     map[string]*Level
	levelsLock sync.RWMutex

	players     map[string]*Player
	playersLock sync.RWMutex
}

func Initialize() gomcc.Plugin {
	db, err := sqlx.Open("sqlite3", "core.db")
	if err != nil {
		log.Println(err)
		return nil
	}

	return &CorePlugin{
		db:      db,
		levels:  make(map[string]*Level),
		players: make(map[string]*Player),
	}
}

func (plugin *CorePlugin) Name() string {
	return "Core"
}

func (plugin *CorePlugin) Enable(server *gomcc.Server) {
	plugin.db.MustExec(`
CREATE TABLE IF NOT EXISTS banned_names(
	name TEXT PRIMARY KEY,
	reason TEXT,
	banned_by TEXT,
	timestamp DATETIME);

CREATE TABLE IF NOT EXISTS banned_ips(
	ip TEXT PRIMARY KEY,
	reason TEXT,
	banned_by TEXT,
	timestamp DATETIME);

CREATE TABLE IF NOT EXISTS levels(
	name TEXT PRIMARY KEY,
	motd TEXT,
	physics INTEGER NOT NULL);

CREATE TABLE IF NOT EXISTS ranks(
	name TEXT PRIMARY KEY,
	permissions TEXT,
	prefix TEXT,
	suffix TEXT,
	is_default INTEGER NOT NULL);

CREATE TABLE IF NOT EXISTS players(
	name TEXT PRIMARY KEY,
	rank TEXT,
	first_login DATETIME,
	last_login DATETIME,
	permissions TEXT,
	nickname TEXT,
	ignore_list TEXT,
	mute INTEGER NOT NULL,
	FOREIGN KEY(rank) REFERENCES ranks(name));`)

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

func (plugin *CorePlugin) Disable(server *gomcc.Server) {
	plugin.db.Close()
}

func (plugin *CorePlugin) addPlayer(player *gomcc.Player) *Player {
	name := player.Name()
	cplayer := &Player{Player: player}

	plugin.db.MustExec(`INSERT OR IGNORE INTO players(name, rank, first_login, mute)
VALUES(?, (SELECT name FROM ranks WHERE is_default = 1), CURRENT_TIMESTAMP, 0);`, name)
	plugin.db.MustExec("UPDATE players SET last_login = CURRENT_TIMESTAMP WHERE name = ?;", name)

	data := struct {
		Nickname   sql.NullString `db:"nickname"`
		IgnoreList sql.NullString `db:"ignore_list"`
		Mute       bool           `db:"mute"`
	}{}
	plugin.db.Get(&data, "SELECT nickname, ignore_list, mute FROM players WHERE name = ?", name)

	cplayer.Mute = data.Mute
	if data.Nickname.Valid {
		player.Nickname = data.Nickname.String
	}
	if data.IgnoreList.Valid && len(data.IgnoreList.String) != 0 {
		cplayer.IgnoreList = strings.Split(data.IgnoreList.String, ",")
	}

	plugin.updatePermissions(cplayer)

	plugin.playersLock.Lock()
	plugin.players[name] = cplayer
	plugin.playersLock.Unlock()
	return cplayer
}

func (plugin *CorePlugin) removePlayer(player *gomcc.Player) {
	plugin.playersLock.Lock()
	delete(plugin.players, player.Name())
	plugin.playersLock.Unlock()
}

func (plugin *CorePlugin) FindPlayer(name string) *Player {
	plugin.playersLock.RLock()
	defer plugin.playersLock.RUnlock()
	return plugin.players[name]
}

func (plugin *CorePlugin) updatePermissions(player *Player) {
	if player.PermGroup == nil {
		player.PermGroup = &gomcc.PermissionGroup{}
		player.AddPermissionGroup(player.PermGroup)
	}

	player.PermGroup.Clear()

	var playerPerms sql.NullString
	row := plugin.db.QueryRow("SELECT Permissions FROM Players WHERE Name = ?", player.Name())
	if err := row.Scan(&playerPerms); err != nil {
		log.Println(err)
	}

	playerData := struct {
		Rank        sql.NullString `db:"rank"`
		Permissions sql.NullString `db:"permissions"`
	}{}
	plugin.db.Get(&playerData, "SELECT rank, permissions FROM players WHERE name = ?", player.Name())

	if playerData.Permissions.Valid {
		for _, perm := range strings.Split(playerData.Permissions.String, ",") {
			player.PermGroup.AddPermission(perm)
		}
	}

	if !playerData.Rank.Valid {
		player.MsgFormat = "%s: &f%s"
		return
	}

	rankData := struct {
		Prefix      sql.NullString `db:"prefix"`
		Suffix      sql.NullString `db:"suffix"`
		Permissions sql.NullString `db:"permissions"`
	}{}
	plugin.db.Get(&rankData, "SELECT prefix, suffix, permissions FROM ranks WHERE name = ?", playerData.Rank.String)

	if rankData.Permissions.Valid {
		for _, perm := range strings.Split(rankData.Permissions.String, ",") {
			player.PermGroup.AddPermission(perm)
		}
	}

	player.MsgFormat = fmt.Sprintf("%s%%s%s: &f%%s", rankData.Prefix.String, rankData.Suffix.String)
}

func (plugin *CorePlugin) addLevel(level *gomcc.Level) *Level {
	name := level.Name
	clevel := &Level{Level: level}

	plugin.db.MustExec("INSERT OR IGNORE INTO levels(name, physics) VALUES(?, 0)", name)

	data := struct {
		MOTD    sql.NullString `db:"motd"`
		Physics bool           `db:"physics"`
	}{}
	plugin.db.Get(&data, "SELECT motd, physics FROM levels WHERE name = ?", name)

	if data.MOTD.Valid {
		level.MOTD = data.MOTD.String
	}

	plugin.disablePhysics(clevel)
	if data.Physics {
		plugin.enablePhysics(clevel)
	}

	plugin.levelsLock.Lock()
	plugin.levels[name] = clevel
	plugin.levelsLock.Unlock()
	return clevel
}

func (plugin *CorePlugin) removeLevel(level *gomcc.Level) {
	plugin.levelsLock.Lock()
	delete(plugin.levels, level.Name)
	plugin.levelsLock.Unlock()
}

func (plugin *CorePlugin) FindLevel(name string) *Level {
	plugin.levelsLock.RLock()
	defer plugin.levelsLock.RUnlock()
	return plugin.levels[name]
}

func (plugin *CorePlugin) handlePlayerLogin(eventType gomcc.EventType, event interface{}) {
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

func (plugin *CorePlugin) handlePlayerJoin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerJoin)
	plugin.addPlayer(e.Player)
}

func (plugin *CorePlugin) handlePlayerQuit(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerQuit)
	plugin.removePlayer(e.Player)
}

func (plugin *CorePlugin) handlePlayerChat(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerChat)
	name := e.Player.Name()
	cplayer := plugin.FindPlayer(name)
	if cplayer.Mute {
		e.Player.SendMessage("You are muted")
		e.Cancel = true
		return
	}

	e.Format = cplayer.MsgFormat
	for i := len(e.Targets) - 1; i >= 0; i-- {
		if plugin.FindPlayer(e.Targets[i].Name()).IsIgnored(name) {
			e.Targets = append(e.Targets[:i], e.Targets[i+1:]...)
		}
	}
}

func (plugin *CorePlugin) handleLevelLoad(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventLevelLoad)
	plugin.addLevel(e.Level)
}

func (plugin *CorePlugin) handleLevelUnload(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventLevelUnload)
	plugin.removeLevel(e.Level)
}
