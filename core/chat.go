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
	"strings"

	"Go-MCC/gomcc"
)

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

	sender.Server().BroadcastMessage("* " + sender.Name() + " " + gomcc.ConvertColors(message))
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
		client := sender.Server().FindClient(args[0])
		if client == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		client.NickName = client.Entity().Name()
		sender.SendMessage("Nick of " + args[0] + " reset")

	case 2:
		if !gomcc.IsValidName(args[1]) {
			sender.SendMessage(args[1] + " is not a valid name")
			return
		}

		client := sender.Server().FindClient(args[0])
		if client == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		client.NickName = args[1]
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
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <message>")
		return
	}

	lastSender := PlayerData(sender.Name()).LastSender
	player := sender.Server().FindClient(lastSender)
	if player == nil {
		sender.SendMessage("Player not found")
		return
	}

	message = gomcc.ConvertColors(message)
	sender.SendMessage("[me -> " + player.Name() + "] " + message)
	player.SendMessage("[" + sender.Name() + " -> me] " + message)

	PlayerData(player.Name()).LastSender = sender.Name()
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

	sender.Server().BroadcastMessage(gomcc.ConvertColors(message))
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

	player := sender.Server().FindClient(args[0])
	if player == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	message = gomcc.ConvertColors(args[1])
	sender.SendMessage("[me -> " + player.Name() + "] " + message)
	player.SendMessage("[" + sender.Name() + " -> me] " + message)

	PlayerData(player.Name()).LastSender = sender.Name()
}
