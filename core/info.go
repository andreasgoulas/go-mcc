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
	"fmt"
	"sort"
	"strings"
	"time"

	"Go-MCC/gomcc"
)

var commandCommands = gomcc.Command{
	Name:        "commands",
	Description: "List all commands.",
	Permission:  "core.commands",
	Handler:     handleCommands,
}

func handleCommands(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) != 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	var cmds []string
	sender.Server().ForEachCommand(func(cmd *gomcc.Command) {
		cmds = append(cmds, cmd.Name)
	})

	sort.Strings(cmds)
	sender.SendMessage(strings.Join(cmds, ", "))
}

var commandHelp = gomcc.Command{
	Name:        "help",
	Description: "Describe a command.",
	Permission:  "core.help",
	Handler:     handleHelp,
}

func handleHelp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <command>")
		return
	}

	cmd := sender.Server().FindCommand(args[0])
	if cmd == nil {
		sender.SendMessage("Unknown command " + args[0])
		return
	}

	sender.SendMessage(cmd.Description)
}

var commandLevels = gomcc.Command{
	Name:        "levels",
	Description: "List all loaded levels.",
	Permission:  "core.levels",
	Handler:     handleLevels,
}

func handleLevels(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) != 0 {
		sender.SendMessage("Usage: " + command.Name)
		return
	}

	var levels []string
	sender.Server().ForEachLevel(func(level *gomcc.Level) {
		levels = append(levels, level.Name())
	})

	sort.Strings(levels)
	sender.SendMessage(strings.Join(levels, ", "))
}

var commandSeen = gomcc.Command{
	Name:        "seen",
	Description: "Check when a player was last online.",
	Permission:  "core.seen",
	Handler:     handleSeen,
}

func fmtDuration(t time.Duration) string {
	t = t.Round(time.Minute)
	d := t / (24 * time.Hour)
	t -= d * (24 * time.Hour)
	h := t / time.Hour
	t -= h * time.Hour
	m := t / time.Minute
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func handleSeen(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	if player := sender.Server().FindPlayer(args[0]); player != nil {
		sender.SendMessage("Player " + args[0] + " is currently online")
		return
	}

	lastLogin, found := CoreDb.LastLogin(args[0])
	if !found {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	dt := time.Now().Sub(lastLogin)
	sender.SendMessage("Player " + args[0] + " was last seen " + fmtDuration(dt) + " ago")
}
