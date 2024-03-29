package main

import (
	"strings"

	"github.com/andreasgoulas/go-mcc/mcc"
)

func (plugin *plugin) handleBack(sender mcc.CommandSender, command *mcc.Command, message string) {
	if _, ok := sender.(*mcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) != 0 {
		command.PrintUsage(sender)
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

func (plugin *plugin) handleSkin(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 2 {
		command.PrintUsage(sender)
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

func (plugin *plugin) handleTp(sender mcc.CommandSender, command *mcc.Command, message string) {
	player, ok := sender.(*mcc.Player)
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
		command.PrintUsage(sender)
		return
	}

	cplayer := plugin.findPlayer(player.Name())
	cplayer.lastLevel = lastLevel
	cplayer.lastLocation = lastLocation
}

func (plugin *plugin) handleSummon(sender mcc.CommandSender, command *mcc.Command, message string) {
	player, ok := sender.(*mcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	if args[0] == "all" {
		player.Level().ForEachEntity(func(entity *mcc.Entity) {
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
