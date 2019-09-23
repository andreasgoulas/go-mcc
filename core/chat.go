// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"log"
	"sort"
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *CorePlugin) PrivateMessage(message string, src, dst gomcc.CommandSender) {
	srcNick := src.Name()
	dstNick := dst.Name()

	var srcPlayer *Player
	if player, ok := src.(*gomcc.Player); ok {
		srcNick = player.Nickname
		srcPlayer = plugin.FindPlayer(src.Name())
		if srcPlayer.Mute {
			src.SendMessage("You are muted")
			return
		}
	}

	var dstPlayer *Player
	if player, ok := dst.(*gomcc.Player); ok {
		dstNick = player.Nickname
		dstPlayer = plugin.FindPlayer(dst.Name())
	}

	src.SendMessage("to " + dstNick + ": &f" + message)
	if dstPlayer != nil {
		if dstPlayer.IsIgnored(src.Name()) {
			return
		} else if srcPlayer != nil {
			dstPlayer.LastSender = src.Name()
		}
	}

	dst.SendMessage("from " + srcNick + ": &f" + message)
}

func (plugin *CorePlugin) BroadcastMessage(src gomcc.CommandSender, message string) {
	log.Printf("%s\n", message)
	src.Server().ForEachPlayer(func(player *gomcc.Player) {
		if !plugin.FindPlayer(player.Name()).IsIgnored(src.Name()) {
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
		player := plugin.FindPlayer(sender.Name())
		if len(player.IgnoreList) == 0 {
			sender.SendMessage("You are not ignoring anyone")
			return
		}

		players := make([]string, len(player.IgnoreList))
		copy(players, player.IgnoreList)
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

		found := false
		player := plugin.FindPlayer(sender.Name())
		for i, name := range player.IgnoreList {
			if name == args[0] {
				found = true
				player.IgnoreList = append(player.IgnoreList[:i], player.IgnoreList[i+1:]...)
				sender.SendMessage("You are no longer ignoring " + args[0])
				break
			}
		}

		if !found {
			player.IgnoreList = append(player.IgnoreList, args[0])
			sender.SendMessage("You are ignoring " + args[0])
		}

		ignoreList := strings.Join(player.IgnoreList, ",")
		plugin.db.MustExec("UPDATE players SET ignore_list = ? WHERE name = ?", ignoreList, sender.Name())

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
		if plugin.FindPlayer(name).Mute {
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

	if player := plugin.FindPlayer(args[0]); player != nil {
		player.Mute = !player.Mute
		if player.Mute {
			sender.SendMessage("Player " + args[0] + " muted")
		} else {
			sender.SendMessage("Player " + args[0] + " unmuted")
		}

		plugin.db.MustExec("UPDATE players SET mute = ? WHERE name = ?", player.Mute, args[0])
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
		plugin.db.MustExec("UPDATE players SET nickname = NULL WHERE name = ?", args[0])
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
		plugin.db.MustExec("UPDATE players SET nickname = ? WHERE name = ?", args[1], args[0])
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

	player := plugin.FindPlayer(sender.Name())
	lastSender := sender.Server().FindPlayer(player.LastSender)
	if lastSender == nil {
		sender.SendMessage("Player not found")
		return
	}

	plugin.PrivateMessage(message, sender, lastSender)
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
