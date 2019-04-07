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

var commandBan = gomcc.Command{
	Name:        "ban",
	Description: "Ban a player from the server.",
	Permission:  "core.ban",
	Handler:     handleBan,
}

func handleBan(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player> <reason>")
		return
	}

	reason := "You have been banned"
	args := strings.SplitN(message, " ", 2)
	if len(args) > 1 {
		reason = args[1]
	}

	BanName(args[0], reason, sender.Name())

	client := sender.Server().FindClient(args[0])
	if client != nil {
		client.Kick(reason)
	}

	sender.SendMessage("Player " + args[0] + " banned")
}

var commandKick = gomcc.Command{
	Name:        "kick",
	Description: "Kick a player from the server.",
	Permission:  "core.kick",
	Handler:     handleKick,
}

func handleKick(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player> <reason>")
		return
	}

	args := strings.SplitN(message, " ", 2)
	player := sender.Server().FindClient(args[0])
	if player == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	reason := "Kicked by " + sender.Name()
	if len(args) > 1 {
		reason = args[1]
	}

	player.Kick(reason)
}

var commandUnban = gomcc.Command{
	Name:        "unban",
	Description: "Remove the ban for a player.",
	Permission:  "core.ban",
	Handler:     handleUnban,
}

func handleUnban(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	UnbanName(args[0])
	sender.SendMessage("Player " + args[0] + " unbanned")
}

func handlePlayerJoin(eventType gomcc.EventType, event interface{}) {
	e := event.(*gomcc.EventPlayerJoin)
	result, reason := IsNameBanned(e.Entity.Name())
	if result {
		e.Cancel = true
		e.CancelReason = reason
	}
}
