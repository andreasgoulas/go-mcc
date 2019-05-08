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

func parseCoord(arg string, curr float64) (float64, error) {
	if strings.HasPrefix(arg, "~") {
		value, err := strconv.Atoi(arg[1:])
		return curr + float64(value), err
	} else {
		value, err := strconv.Atoi(arg)
		return float64(value), err
	}
}

var commandBack = gomcc.Command{
	Name:        "back",
	Description: "Return to your location before your last teleportation.",
	Permission:  "core.back",
	Handler:     handleBack,
}

func handleBack(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) != 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	CorePlayers.Lock.RLock()
	defer CorePlayers.Lock.RUnlock()

	data := CorePlayers.Online[sender.Name()]
	if data.LastLevel == nil {
		sender.SendMessage("Location not found")
		return
	}

	player.TeleportLevel(data.LastLevel)
	player.Teleport(data.LastLocation)
}

var commandSkin = gomcc.Command{
	Name:        "skin",
	Description: "Set the skin of a player.",
	Permission:  "core.skin",
	Handler:     handleSkin,
}

func handleSkin(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 2 {
		sender.SendMessage("Usage: " + command.Name + " <player> <skin>")
		return
	}

	entity := sender.Server().FindEntity(args[0])
	if entity == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	entity.SkinName = args[1]
	entity.Respawn()
	sender.SendMessage("Skin of " + args[0] + " set to " + args[1])
}

var commandTp = gomcc.Command{
	Name:        "tp",
	Description: "Teleport to another player.",
	Permission:  "core.tp",
	Handler:     handleTp,
}

func handleTp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	lastLevel := player.Level()
	lastLocation := player.Location()

	args := strings.Fields(message)
	switch len(args) {
	case 1:
		target := sender.Server().FindEntity(args[0])
		if target == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		player.TeleportLevel(target.Level())
		player.Teleport(target.Location())

	case 3:
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

	default:
		sender.SendMessage("Usage: " + command.Name + " <player> or <x> <y> <z>")
		return
	}

	CorePlayers.Lock.RLock()
	defer CorePlayers.Lock.RUnlock()

	data := CorePlayers.Online[player.Name()]
	data.LastLevel = lastLevel
	data.LastLocation = lastLocation
}

var commandSummon = gomcc.Command{
	Name:        "summon",
	Description: "Summon a player to your location.",
	Permission:  "core.summon",
	Handler:     handleSummon,
}

func handleSummon(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	player, ok := sender.(*gomcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player> or all")
		return
	}

	if args[0] == "all" {
		player.Level().ForEachEntity(func(entity *gomcc.Entity) {
			entity.Teleport(player.Location())
		})
	} else {
		target := sender.Server().FindEntity(args[0])
		if target == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		level := player.Level()
		if level != target.Level() {
			target.TeleportLevel(level)
		}

		target.Teleport(player.Location())
	}
}
