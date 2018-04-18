// Copyright 2017-2018 Andrew Goulas
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

var CommandMe = gomcc.Command{
	Name:        "me",
	Description: "Broadcast an action.",
	Permission:  "core.me",
	Handler:     HandleMe,
}

func HandleMe(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <action>")
		return
	}

	sender.Server().BroadcastMessage("* " + sender.Name() + " " + gomcc.ConvertColors(message))
}

var CommandSay = gomcc.Command{
	Name:        "say",
	Description: "Broadcast a message.",
	Permission:  "core.say",
	Handler:     HandleSay,
}

func HandleSay(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <message>")
		return
	}

	sender.Server().BroadcastMessage(gomcc.ConvertColors(message))
}

var CommandTell = gomcc.Command{
	Name:        "tell",
	Description: "Send a private message to a player.",
	Permission:  "core.tell",
	Handler:     HandleTell,
}

func HandleTell(sender gomcc.CommandSender, command *gomcc.Command, message string) {
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
}
