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

package main

import (
	"sort"
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *CorePlugin) PrivateMessage(message string, src, dst gomcc.CommandSender) {
	var psrc, pdst bool

	srcName := src.Name()
	player, psrc := src.(*gomcc.Player)
	if psrc {
		if plugin.Players.Player(src.Name()).Mute {
			src.SendMessage("You are muted")
			return
		}

		srcName = player.Nickname
	}

	dstName := dst.Name()
	player, pdst = dst.(*gomcc.Player)
	if pdst {
		dstName = player.Nickname
	}

	if psrc && pdst {
		plugin.Players.Player(dst.Name()).LastSender = src.Name()
	}

	src.SendMessage("to " + dstName + ": &f" + message)
	if pdst && plugin.Players.Player(dst.Name()).IsIgnored(src.Name()) {
		return
	}

	dst.SendMessage("from " + srcName + ": &f" + message)
}

func (plugin *CorePlugin) BroadcastMessage(src gomcc.CommandSender, message string) {
	src.Server().ForEachPlayer(func(player *gomcc.Player) {
		if !plugin.Players.Player(player.Name()).IsIgnored(src.Name()) {
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
		cplayer := plugin.Players.Player(sender.Name())
		if len(cplayer.Ignore) == 0 {
			sender.SendMessage("You are not ignoring anyone")
			return
		}

		var players []string
		for _, player := range cplayer.Ignore {
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

		cplayer := plugin.Players.Player(sender.Name())
		for i, player := range cplayer.Ignore {
			if player == args[0] {
				cplayer.Ignore = append(cplayer.Ignore[:i], cplayer.Ignore[i+1:]...)
				sender.SendMessage("You are no longer ignoring " + args[0])
				return
			}
		}

		cplayer.Ignore = append(cplayer.Ignore, args[0])
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
		if plugin.Players.Player(name).Mute {
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

	cplayer := plugin.Players.OfflinePlayer(args[0])
	cplayer.Mute = !cplayer.Mute
	if cplayer.Mute {
		sender.SendMessage("Player " + args[0] + " muted")
	} else {
		sender.SendMessage("Player " + args[0] + " unmuted")
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
		plugin.Players.Player(player.Name()).Nickname = ""
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
		plugin.Players.Player(player.Name()).Nickname = args[1]
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

	lastSender := plugin.Players.Player(sender.Name()).LastSender
	player := sender.Server().FindPlayer(lastSender)
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
