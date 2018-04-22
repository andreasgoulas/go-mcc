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

var CommandGoto = gomcc.Command{
	Name:        "goto",
	Description: "Move to another level.",
	Permission:  "core.goto",
	Handler:     HandleGoto,
}

func HandleGoto(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	client, ok := sender.(*gomcc.Client)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <map>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Map " + args[0] + " not found")
		return
	}

	if level == client.Entity.Level {
		sender.SendMessage("You are already in " + level.Name)
		return
	}

	client.Entity.TeleportLevel(level)
}

var CommandLoad = gomcc.Command{
	Name:        "load",
	Description: "Load a level.",
	Permission:  "core.load",
	Handler:     HandleLoad,
}

func HandleLoad(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <map>")
		return
	}

	_, err := sender.Server().LoadLevel(args[0])
	if err != nil {
		sender.SendMessage("Could not load map " + args[0])
		return
	}

	sender.SendMessage("Map " + args[0] + " loaded")
}

var CommandMain = gomcc.Command{
	Name:        "main",
	Description: "Set the main level.",
	Permission:  "core.main",
	Handler:     HandleMain,
}

func HandleMain(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Main level is " + sender.Server().MainLevel.Name)
		return
	}

	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <map>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Map " + args[0] + " not found")
		return
	}

	sender.Server().MainLevel = level
	sender.SendMessage("Set main level to " + level.Name)
}

var CommandSpawn = gomcc.Command{
	Name:        "spawn",
	Description: "Teleport to the spawn location of the level.",
	Permission:  "core.spawn",
	Handler:     HandleSpawn,
}

func HandleSpawn(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	client, ok := sender.(*gomcc.Client)
	if !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) > 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	client.Entity.Teleport(client.Entity.Level.Spawn)
}

var CommandUnload = gomcc.Command{
	Name:        "unload",
	Description: "Unload a level.",
	Permission:  "core.unload",
	Handler:     HandleUnload,
}

func HandleUnload(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <map>")
		return
	}

	level := sender.Server().FindLevel(args[0])
	if level == nil {
		sender.SendMessage("Map " + args[0] + " not found")
		return
	}

	if level == sender.Server().MainLevel {
		sender.SendMessage("Map " + args[0] + " is the main level")
		return
	}

	sender.Server().UnloadLevel(level)
	sender.SendMessage("Map " + args[0] + " unloaded")
}
