// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *Plugin) handleBack(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if _, ok := sender.(*gomcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) != 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	player := plugin.findPlayer(sender.Name())
	if player.lastLevel == nil {
		sender.SendMessage("Location not found")
		return
	}

	player.TeleportLevel(player.lastLevel)
	player.Teleport(player.lastLocation)
}

func (plugin *Plugin) handleSkin(sender gomcc.CommandSender, command *gomcc.Command, message string) {
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

func (plugin *Plugin) handleTp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
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

	cplayer := plugin.findPlayer(player.Name())
	cplayer.lastLevel = lastLevel
	cplayer.lastLocation = lastLocation
}

func (plugin *Plugin) handleSummon(sender gomcc.CommandSender, command *gomcc.Command, message string) {
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
