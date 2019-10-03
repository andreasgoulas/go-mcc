// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/structinf/go-mcc/mcc"
)

func (plugin *Plugin) handleCopyLvl(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 2 {
		command.PrintUsage(sender)
		return
	}

	src := sender.Server().FindLevel(args[0])
	if src == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	dest := sender.Server().FindLevel(args[1])
	if dest != nil {
		sender.SendMessage("Level " + args[1] + " already exists")
		return
	}

	dest = src.Clone(args[1])
	sender.Server().AddLevel(dest)
	sender.SendMessage("Level " + args[0] + " has been copied to " + args[1])
}

func (plugin *Plugin) handleEnv(sender mcc.CommandSender, command *mcc.Command, message string) {
	player, ok := sender.(*mcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	level := player.Level()
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		if args[0] == "reset" {
			level.EnvConfig = level.DefaultEnvConfig()
			level.SendEnvConfig(mcc.EnvPropAll)
			return
		}

	case 2:
		switch mask := envOption(args[0], args[1], &level.EnvConfig); mask {
		case 0:
			sender.SendMessage("Unknown option")
		case -1:
			sender.SendMessage("Invalid value")
		default:
			level.SendEnvConfig(uint32(mask))
		}

		return
	}

	command.PrintUsage(sender)
	return
}

func (plugin *Plugin) handleGoto(sender mcc.CommandSender, command *mcc.Command, message string) {
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

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	if level == player.Level() {
		sender.SendMessage("You are already in " + level.Name)
		return
	}

	player.TeleportLevel(level)
}

func (plugin *Plugin) handleLoad(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	_, err := sender.Server().LoadLevel(args[0])
	if err != nil {
		sender.SendMessage("Could not load level " + args[0])
		return
	}

	sender.SendMessage("Level " + args[0] + " loaded")
}

func (plugin *Plugin) handleMain(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 0:
		sender.SendMessage("Main level is " + sender.Server().MainLevel.Name)

	case 1:
		level := sender.Server().FindLevel(args[0])
		if level == nil {
			sender.SendMessage("Level " + args[0] + " not found")
			return
		}

		sender.Server().MainLevel = level
		sender.SendMessage("Set main level to " + level.Name)

	default:
		command.PrintUsage(sender)
	}
}

func (plugin *Plugin) handleNewLvl(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) < 5 {
		command.PrintUsage(sender)
		return
	}

	width, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		sender.SendMessage(args[1] + " is not a valid number")
		return
	}

	height, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		sender.SendMessage(args[2] + " is not a valid number")
		return
	}

	length, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		sender.SendMessage(args[3] + " is not a valid number")
		return
	}

	genFunc, ok := mcc.Generators[args[4]]
	if !ok {
		sender.SendMessage("Generator " + args[4] + " not found")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level != nil {
		sender.SendMessage("Level " + args[0] + " already exists")
		return
	}

	level = mcc.NewLevel(args[0], int(width), int(height), int(length))
	if level == nil {
		sender.SendMessage("Could not create level")
		return
	}

	generator := genFunc(args[5:]...)
	generator.Generate(level)

	sender.Server().AddLevel(level)
	sender.SendMessage("Level " + level.Name + " created")
}

func (plugin *Plugin) handlePhysics(sender mcc.CommandSender, command *mcc.Command, message string) {
	var level *level
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		if player, ok := sender.(*mcc.Player); !ok {
			sender.SendMessage("You are not a player")
			return
		} else {
			level = plugin.findLevel(player.Level().Name)
		}

	case 2:
		level = plugin.findLevel(args[0])
		if level == nil {
			sender.SendMessage("Level " + args[0] + " not found")
			return
		}

		args = args[1:]

	default:
		command.PrintUsage(sender)
		return
	}

	if value, err := strconv.ParseBool(args[0]); err != nil {
		sender.SendMessage(args[0] + " is not a valid boolean")
		return
	} else {
		if value != level.physics {
			level.physics = value
			if value {
				plugin.enablePhysics(level)
			} else {
				plugin.disablePhysics(level)
			}
		}

		sender.SendMessage(fmt.Sprintf("Physics set to %t", value))
	}
}

func (plugin *Plugin) handleSave(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	if args[0] == "all" {
		sender.Server().ForEachLevel(func(level *mcc.Level) {
			sender.Server().SaveLevel(level)
		})
		sender.SendMessage("All levels have been saved")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	sender.Server().SaveLevel(level)
	sender.SendMessage("Level " + level.Name + " saved")
}

func (plugin *Plugin) handleSetSpawn(sender mcc.CommandSender, command *mcc.Command, message string) {
	player, ok := sender.(*mcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	switch len(args) {
	case 0:
		level := player.Level()
		level.Spawn = player.Location()
		level.Dirty = true

		player.SetSpawn()
		sender.SendMessage("Spawn location set to your current location")

	case 1:
		target := sender.Server().FindPlayer(args[0])
		if target == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		if target.Level() != player.Level() {
			sender.SendMessage(target.Name() + " is on a different level")
			return
		}

		target.Teleport(player.Location())
		target.SetSpawn()
		sender.SendMessage("Spawn location of " + player.Name() + " set to your current location")

	default:
		command.PrintUsage(sender)
	}
}

func (plugin *Plugin) handleSpawn(sender mcc.CommandSender, command *mcc.Command, message string) {
	player, ok := sender.(*mcc.Player)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) > 0 {
		command.PrintUsage(sender)
		return
	}

	player.Teleport(player.Level().Spawn)
}

func (plugin *Plugin) handleUnload(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	if level == sender.Server().MainLevel {
		sender.SendMessage("Level " + args[0] + " is the main level")
		return
	}

	sender.Server().UnloadLevel(level)
	sender.SendMessage("Level " + args[0] + " unloaded")
}
