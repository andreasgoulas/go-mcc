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
	"strconv"
	"strings"

	"Go-MCC/gomcc"
)

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

var commandSkin = gomcc.Command{
	Name:        "skin",
	Description: "Set the skin of a player.",
	Permission:  "core.skin",
	Handler:     handleSkin,
}

func handleSkin(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 2 {
		sender.SendMessage("Usage: " + command.Name + " <name> <skin>")
		return
	}

	entity := sender.Server().FindEntity(args[0])
	if entity == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	entity.SkinName = args[1]
	entity.Respawn()

	if entity.Client == sender {
		sender.SendMessage("Skin set to " + args[1])
	} else {
		sender.SendMessage("Skin of " + args[0] + " set to " + args[1])
	}
}

var commandTp = gomcc.Command{
	Name:        "tp",
	Description: "Teleport to another player.",
	Permission:  "core.tp",
	Handler:     handleTp,
}

func parseCoord(arg string, curr float64) (float64, error) {
	if strings.HasPrefix(arg, "~") {
		value, err := strconv.Atoi(arg[1:])
		return curr + float64(value), err
	} else {
		value, err := strconv.Atoi(arg)
		return float64(value), err
	}
}

func handleTp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	client, ok := sender.(*gomcc.Client)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	player := client.Entity
	args := strings.Split(message, " ")
	if len(args) == 1 && len(args[0]) > 0 {
		entity := sender.Server().FindEntity(args[0])
		if entity == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		level := entity.Level()
		if level != player.Level() {
			player.TeleportLevel(level)
		}

		player.Teleport(entity.Location())
	} else if len(args) == 3 {
		var err error
		location := player.Location()

		location.X, err = parseCoord(args[0], location.X)
		if err != nil {
			sender.SendMessage(args[0] + " is not a valid number")
			return
		}

		location.Y, err = parseCoord(args[1], location.Y)
		if err != nil {
			sender.SendMessage(args[1] + " is not a valid number")
			return
		}

		location.Z, err = parseCoord(args[2], location.Z)
		if err != nil {
			sender.SendMessage(args[2] + " is not a valid number")
			return
		}

		player.Teleport(location)
	} else {
		sender.SendMessage("Usage: " + command.Name + " <player> or <x> <y> <z>")
	}
}
