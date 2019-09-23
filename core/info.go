// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/structinf/Go-MCC/gomcc"
)

func fmtDuration(t time.Duration) string {
	t = t.Round(time.Minute)
	d := t / (24 * time.Hour)
	t -= d * (24 * time.Hour)
	h := t / time.Hour
	t -= h * time.Hour
	m := t / time.Minute
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func (plugin *CorePlugin) handleCommands(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) != 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	var cmds []string
	sender.Server().ForEachCommand(func(cmd *gomcc.Command) {
		cmds = append(cmds, cmd.Name)
	})

	sort.Strings(cmds)
	sender.SendMessage(strings.Join(cmds, ", "))
}

func (plugin *CorePlugin) handleHelp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <command>")
		return
	}

	cmd := sender.Server().FindCommand(args[0])
	if cmd == nil {
		sender.SendMessage("Unknown command " + args[0])
		return
	}

	sender.SendMessage(cmd.Description)
}

func (plugin *CorePlugin) handleLevels(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) != 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	var levels []string
	sender.Server().ForEachLevel(func(level *gomcc.Level) {
		levels = append(levels, level.Name)
	})

	sort.Strings(levels)
	sender.SendMessage(strings.Join(levels, ", "))
}

func (plugin *CorePlugin) handlePlayers(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	var players []string
	args := strings.Fields(message)
	switch len(args) {
	case 0:
		sender.Server().ForEachPlayer(func(player *gomcc.Player) {
			players = append(players, player.Name())
		})

	case 1:
		level := sender.Server().FindLevel(args[0])
		if level == nil {
			sender.SendMessage("Level " + args[0] + " not found")
			return
		}

		level.ForEachPlayer(func(player *gomcc.Player) {
			players = append(players, player.Name())
		})

	default:
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	sort.Strings(players)
	sender.SendMessage(strings.Join(players, ", "))
}

func (plugin *CorePlugin) handleSeen(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	if sender.Server().FindPlayer(args[0]) != nil {
		sender.SendMessage("Player " + args[0] + " is currently online")
		return
	}

	var lastLogin time.Time
	if plugin.db.Get(&lastLogin, "SELECT last_login FROM players WHERE name = ?", args[0]) == nil {
		dt := time.Now().Sub(lastLogin)
		sender.SendMessage("Player " + args[0] + " was last seen " + fmtDuration(dt) + " ago")
	} else {
		sender.SendMessage("Player " + args[0] + " not found")
	}
}
