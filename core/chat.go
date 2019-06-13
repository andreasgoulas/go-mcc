// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"sort"
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *CorePlugin) PrivateMessage(message string, src, dst gomcc.CommandSender) {
	srcNick := src.Name()
	dstNick := dst.Name()

	var srcInfo, dstInfo *PlayerInfo
	if player, ok := src.(*gomcc.Player); ok {
		srcNick = player.Nickname
		srcInfo = plugin.Players.Find(src.Name())
		if srcInfo.Mute {
			src.SendMessage("You are muted")
			return
		}
	}

	if player, ok := dst.(*gomcc.Player); ok {
		dstNick = player.Nickname
		dstInfo = plugin.Players.Find(dst.Name())
		if srcInfo != nil {
			dstInfo.Player.LastSender = src.Name()
		}
	}

	src.SendMessage("to " + dstNick + ": &f" + message)
	if dstInfo != nil && dstInfo.IsIgnored(src.Name()) {
		return
	}

	dst.SendMessage("from " + srcNick + ": &f" + message)
}

func (plugin *CorePlugin) BroadcastMessage(src gomcc.CommandSender, message string) {
	src.Server().ForEachPlayer(func(player *gomcc.Player) {
		if !plugin.Players.Find(player.Name()).IsIgnored(src.Name()) {
			player.SendMessage(message)
		}
	})
}

func (plugin *CorePlugin) handleIgnore(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if _, ok := sender.(*gomcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	switch len(args) {
	case 0:
		info := plugin.Players.Find(sender.Name())
		if len(info.Ignore) == 0 {
			sender.SendMessage("You are not ignoring anyone")
			return
		}

		var players []string
		for _, player := range info.Ignore {
			players = append(players, player)
		}

		sort.Strings(players)
		sender.SendMessage(strings.Join(players, ", "))

	case 1:
		if !gomcc.IsValidName(args[0]) {
			sender.SendMessage(args[0] + " is not a valid name")
			return
		}

		if args[0] == sender.Name() {
			sender.SendMessage("You cannot ignore yourself")
			return
		}

		info := plugin.Players.Find(sender.Name())
		for i, player := range info.Ignore {
			if player == args[0] {
				info.Ignore = append(info.Ignore[:i], info.Ignore[i+1:]...)
				sender.SendMessage("You are no longer ignoring " + args[0])
				return
			}
		}

		info.Ignore = append(info.Ignore, args[0])
		sender.SendMessage("You are ignoring " + args[0])

	default:
		sender.SendMessage("Usage: " + command.Name + " <player>")
	}
}

func (plugin *CorePlugin) handleMe(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <action>")
		return
	}

	name := sender.Name()
	if player, ok := sender.(*gomcc.Player); ok {
		if plugin.Players.Find(name).Mute {
			sender.SendMessage("You are muted")
			return
		}

		name = player.Nickname
	}

	plugin.BroadcastMessage(sender, "* "+name+" "+message)
}

func (plugin *CorePlugin) handleMute(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	if info := plugin.Players.Find(args[0]); info != nil {
		info.Mute = !info.Mute
		if info.Mute {
			sender.SendMessage("Player " + args[0] + " muted")
		} else {
			sender.SendMessage("Player " + args[0] + " unmuted")
		}
	} else {
		sender.SendMessage("Player " + args[0] + " not found")
	}
}

func (plugin *CorePlugin) handleNick(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		player := sender.Server().FindPlayer(args[0])
		if player == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		player.Nickname = player.Name()
		plugin.Players.Find(player.Name()).Nickname = ""
		sender.SendMessage("Nick of " + args[0] + " reset")

	case 2:
		if !gomcc.IsValidName(args[1]) {
			sender.SendMessage(args[1] + " is not a valid name")
			return
		}

		player := sender.Server().FindPlayer(args[0])
		if player == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		player.Nickname = args[1]
		plugin.Players.Find(player.Name()).Nickname = args[1]
		sender.SendMessage("Nick of " + args[0] + " set to " + args[1])

	default:
		sender.SendMessage("Usage: " + command.Name + " <player> <nick>")
	}
}

func (plugin *CorePlugin) handleR(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if _, ok := sender.(*gomcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <message>")
		return
	}

	info := plugin.Players.Find(sender.Name())
	player := sender.Server().FindPlayer(info.Player.LastSender)
	if player == nil {
		sender.SendMessage("Player not found")
		return
	}

	plugin.PrivateMessage(message, sender, player)
}

func (plugin *CorePlugin) handleSay(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <message>")
		return
	}

	sender.Server().BroadcastMessage(message)
}

func (plugin *CorePlugin) handleTell(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.SplitN(message, " ", 2)
	if len(args) < 2 {
		sender.SendMessage("Usage: " + command.Name + " <player> <message>")
		return
	}

	player := sender.Server().FindPlayer(args[0])
	if player == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	plugin.PrivateMessage(args[1], sender, player)
}
