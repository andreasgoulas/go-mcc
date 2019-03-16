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

var commandGoto = gomcc.Command{
	Name:        "goto",
	Description: "Move to another level.",
	Permission:  "core.goto",
	Handler:     handleGoto,
}

func handleGoto(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	client, ok := sender.(*gomcc.Client)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	if level == client.Entity.Level() {
		sender.SendMessage("You are already in " + level.Name)
		return
	}

	client.Entity.TeleportLevel(level)
}

var commandLoad = gomcc.Command{
	Name:        "load",
	Description: "Load a level.",
	Permission:  "core.load",
	Handler:     handleLoad,
}

func handleLoad(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	_, err := sender.Server().LoadLevel(args[0])
	if err != nil {
		sender.SendMessage("Could not load level " + args[0])
		return
	}

	sender.SendMessage("Level " + args[0] + " loaded")
}

var commandMain = gomcc.Command{
	Name:        "main",
	Description: "Set the main level.",
	Permission:  "core.main",
	Handler:     handleMain,
}

func handleMain(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Main level is " + sender.Server().MainLevel.Name)
		return
	}

	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Level " + args[0] + " not found")
		return
	}

	sender.Server().MainLevel = level
	sender.SendMessage("Set main level to " + level.Name)
}

var commandNewLvl = gomcc.Command{
	Name:        "newlvl",
	Description: "Create a new level.",
	Permission:  "core.newlvl",
	Handler:     handleNewLvl,
}

func handleNewLvl(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 5 {
		sender.SendMessage("Usage: " + command.Name + " <name> <width> <height> <length> <theme>")
		return
	}

	width, err := strconv.ParseUint(args[1], 10, 0)
	if err != nil {
		sender.SendMessage(args[1] + " is not a valid number")
		return
	}

	height, err := strconv.ParseUint(args[2], 10, 0)
	if err != nil {
		sender.SendMessage(args[2] + " is not a valid number")
		return
	}

	length, err := strconv.ParseUint(args[3], 10, 0)
	if err != nil {
		sender.SendMessage(args[3] + " is not a valid number")
		return
	}

	generator, ok := gomcc.Generators[args[4]]
	if !ok {
		sender.SendMessage("Generator " + args[4] + " not found")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level != nil {
		sender.SendMessage("Level " + args[0] + " already exists")
		return
	}

	level = gomcc.NewLevel(args[0], uint(width), uint(height), uint(length))
	if level == nil {
		sender.SendMessage("Could not create level")
		return
	}

	generator.Generate(level)

	sender.Server().AddLevel(level)
	sender.SendMessage("Level " + level.Name + " created")
}

var commandSave = gomcc.Command{
	Name:        "save",
	Description: "Save a level.",
	Permission:  "core.save",
	Handler:     handleSave,
}

func handleSave(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
		return
	}

	if args[0] == "all" {
		sender.Server().ForEachLevel(func(level *gomcc.Level) {
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

var commandSetSpawn = gomcc.Command{
	Name:        "setspawn",
	Description: "Set the spawn location of the level to your location.",
	Permission:  "core.setspawn",
	Handler:     handleSetSpawn,
}

func handleSetSpawn(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	client, ok := sender.(*gomcc.Client)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) == 0 {
		client.Entity.Level().Spawn = client.Entity.Location()
		client.SetSpawn()
		sender.SendMessage("Spawn location set to your current location")
		return
	}

	args := strings.Split(message, " ")
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	player := sender.Server().FindClient(args[0])
	if player == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	if player.Entity.Level() != client.Entity.Level() {
		sender.SendMessage(player.Name() + " is on a different level")
		return
	}

	player.Entity.Teleport(client.Entity.Location())
	player.SetSpawn()
	sender.SendMessage("Spawn location of " + player.Name() + " set to your current location")
}

var commandSpawn = gomcc.Command{
	Name:        "spawn",
	Description: "Teleport to the spawn location of the level.",
	Permission:  "core.spawn",
	Handler:     handleSpawn,
}

func handleSpawn(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	client, ok := sender.(*gomcc.Client)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) > 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	level := client.Entity.Level()
	client.Entity.Teleport(level.Spawn)
}

var commandUnload = gomcc.Command{
	Name:        "unload",
	Description: "Unload a level.",
	Permission:  "core.unload",
	Handler:     handleUnload,
}

func handleUnload(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <level>")
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
