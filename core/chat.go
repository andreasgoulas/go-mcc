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
	"sort"
	"strings"

	"Go-MCC/gomcc"
)

func SendPm(message string, src, dst gomcc.CommandSender) {
	var psrc, pdst bool

	srcName := src.Name()
	player, psrc := src.(*gomcc.Player)
	if psrc {
		if CorePlayers.Player(src.Name()).Mute {
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
		CorePlayers.Player(dst.Name()).LastSender = src.Name()
	}

	src.SendMessage("to " + dstName + ": &f" + message)
	if pdst && CorePlayers.Player(dst.Name()).IsIgnored(src.Name()) {
		return
	}

	dst.SendMessage("from " + srcName + ": &f" + message)
}

func Broadcast(src gomcc.CommandSender, message string) {
	src.Server().ForEachPlayer(func(player *gomcc.Player) {
		if !CorePlayers.Player(player.Name()).IsIgnored(src.Name()) {
			player.SendMessage(message)
		}
	})
}

var commandIgnore = gomcc.Command{
	Name:        "ignore",
	Description: "Ignore chat from a player",
	Permission:  "core.ignore",
	Handler:     handleIgnore,
}

func handleIgnore(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if _, ok := sender.(*gomcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	switch len(args) {
	case 0:
		cplayer := CorePlayers.Player(sender.Name())
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

		cplayer := CorePlayers.Player(sender.Name())
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

var commandMe = gomcc.Command{
	Name:        "me",
	Description: "Broadcast an action.",
	Permission:  "core.me",
	Handler:     handleMe,
}

func handleMe(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <action>")
		return
	}

	name := sender.Name()
	if player, ok := sender.(*gomcc.Player); ok {
		if CorePlayers.Player(name).Mute {
			sender.SendMessage("You are muted")
			return
		}

		name = player.Nickname
	}

	Broadcast(sender, "* "+name+" "+message)
}

var commandMute = gomcc.Command{
	Name:        "mute",
	Description: "Mute a player.",
	Permission:  "core.mute",
	Handler:     handleMute,
}

func handleMute(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	cplayer := CorePlayers.OfflinePlayer(args[0])
	cplayer.Mute = !cplayer.Mute
	if cplayer.Mute {
		sender.SendMessage("Player " + args[0] + " muted")
	} else {
		sender.SendMessage("Player " + args[0] + " unmuted")
	}
}

var commandNick = gomcc.Command{
	Name:        "nick",
	Description: "Set the nickname of a player",
	Permission:  "core.nick",
	Handler:     handleNick,
}

func handleNick(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		player := sender.Server().FindPlayer(args[0])
		if player == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		CorePlayers.Lock.RLock()
		defer CorePlayers.Lock.RUnlock()

		player.Nickname = player.Name()
		CorePlayers.Players[player.Name()].Nickname = ""
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

		CorePlayers.Lock.RLock()
		defer CorePlayers.Lock.RUnlock()

		player.Nickname = args[1]
		CorePlayers.Players[player.Name()].Nickname = args[1]
		sender.SendMessage("Nick of " + args[0] + " set to " + args[1])

	default:
		sender.SendMessage("Usage: " + command.Name + " <player> <nick>")
	}
}

var commandR = gomcc.Command{
	Name:        "r",
	Description: "Reply to the last message.",
	Permission:  "core.r",
	Handler:     handleR,
}

func handleR(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if _, ok := sender.(*gomcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <message>")
		return
	}

	lastSender := CorePlayers.Player(sender.Name()).LastSender
	player := sender.Server().FindPlayer(lastSender)
	if player == nil {
		sender.SendMessage("Player not found")
		return
	}

	SendPm(message, sender, player)
}

var commandSay = gomcc.Command{
	Name:        "say",
	Description: "Broadcast a message.",
	Permission:  "core.say",
	Handler:     handleSay,
}

func handleSay(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <message>")
		return
	}

	sender.Server().BroadcastMessage(message)
}

var commandTell = gomcc.Command{
	Name:        "tell",
	Description: "Send a private message to a player.",
	Permission:  "core.tell",
	Handler:     handleTell,
}

func handleTell(sender gomcc.CommandSender, command *gomcc.Command, message string) {
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

	SendPm(args[1], sender, player)
}
