package main

import (
	"sort"
	"strings"
	"time"

	"github.com/andreasgoulas/go-mcc/mcc"
)

func (plugin *plugin) handleCommands(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) != 0 {
		command.PrintUsage(sender)
		return
	}

	var cmds []string
	sender.Server().ForEachCommand(func(cmd *mcc.Command) {
		cmds = append(cmds, cmd.Name)
	})

	sort.Strings(cmds)
	sender.SendMessage(strings.Join(cmds, ", "))
}

func (plugin *plugin) handleHelp(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	cmd := sender.Server().FindCommand(args[0])
	if cmd == nil {
		sender.SendMessage("Unknown command " + args[0])
		return
	}

	sender.SendMessage(cmd.Description)
	cmd.PrintUsage(sender)
}

func (plugin *plugin) handleLevels(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) != 0 {
		command.PrintUsage(sender)
		return
	}

	var levels []string
	sender.Server().ForEachLevel(func(level *mcc.Level) {
		levels = append(levels, level.Name)
	})

	sort.Strings(levels)
	sender.SendMessage(strings.Join(levels, ", "))
}

func (plugin *plugin) handlePlayers(sender mcc.CommandSender, command *mcc.Command, message string) {
	var players []string
	args := strings.Fields(message)
	switch len(args) {
	case 0:
		sender.Server().ForEachPlayer(func(player *mcc.Player) {
			players = append(players, player.Name())
		})

	case 1:
		level := sender.Server().FindLevel(args[0])
		if level == nil {
			sender.SendMessage("Level " + args[0] + " not found")
			return
		}

		level.ForEachPlayer(func(player *mcc.Player) {
			players = append(players, player.Name())
		})

	default:
		command.PrintUsage(sender)
		return
	}

	sort.Strings(players)
	sender.SendMessage(strings.Join(players, ", "))
}

func (plugin *plugin) handleSeen(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	if sender.Server().FindPlayer(args[0]) != nil {
		sender.SendMessage("Player " + args[0] + " is currently online")
		return
	}

	if db, ok := plugin.db.queryPlayer(args[0]); ok {
		dt := time.Since(db.LastLogin)
		sender.SendMessage("Player " + args[0] + " was last seen " + fmtDuration(dt) + " ago")
	} else {
		sender.SendMessage("Player " + args[0] + " not found")
	}
}
