package main

import (
	"net"
	"strings"

	"github.com/AndreasGoulas/go-mcc/mcc"
)

func (plugin *plugin) handleBan(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) == 0 {
		command.PrintUsage(sender)
		return
	}

	reason := "You have been banned"
	args := strings.SplitN(message, " ", 2)
	if len(args) > 1 {
		reason = args[1]
	}

	if !mcc.IsValidName(args[0]) {
		sender.SendMessage(args[0] + " is not a valid name")
		return
	}

	plugin.db.ban(args[0], reason, sender.Name())
	if player := sender.Server().FindPlayer(args[0]); player != nil {
		player.Kick(reason)
	}

	sender.SendMessage("Player " + args[0] + " banned")
}

func (plugin *plugin) handleBanIp(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) == 0 {
		command.PrintUsage(sender)
		return
	}

	reason := "You have been banned"
	args := strings.SplitN(message, " ", 2)
	if len(args) > 1 {
		reason = args[1]
	}

	if net.ParseIP(args[0]) == nil {
		sender.SendMessage(args[0] + " is not a valid IP address")
		return
	}

	plugin.db.banIP(args[0], reason, sender.Name())
	sender.Server().ForEachPlayer(func(player *mcc.Player) {
		if player.RemoteAddr() == args[0] {
			player.Kick(reason)
		}
	})

	sender.SendMessage("IP " + args[0] + " banned")
}

func (plugin *plugin) handleKick(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) == 0 {
		command.PrintUsage(sender)
		return
	}

	args := strings.SplitN(message, " ", 2)
	player := sender.Server().FindPlayer(args[0])
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

func (plugin *plugin) handleRank(sender mcc.CommandSender, command *mcc.Command, message string) {
	var rank *mcc.Rank
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		rank = nil

	case 2:
		if rank = plugin.findRank(args[1]); rank == nil {
			sender.SendMessage("Rank " + args[1] + " not found")
			return
		}

	default:
		command.PrintUsage(sender)
		return
	}

	if player := plugin.findPlayer(args[0]); player == nil {
		sender.SendMessage("Player " + args[0] + " not found")
	} else {
		player.Rank = rank
		player.SendPermissions()
		if rank == nil {
			sender.SendMessage("Rank of " + args[0] + " reset")
		} else {
			sender.SendMessage("Rank of " + args[0] + " set to " + args[1])
		}
	}
}

func (plugin *plugin) handleUnban(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	if plugin.db.unban(args[0]) {
		sender.SendMessage("Player " + args[0] + " unbanned")
	} else {
		sender.SendMessage("Player " + args[0] + " is not banned")
	}
}

func (plugin *plugin) handleUnbanIp(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	if plugin.db.unbanIP(args[0]) {
		sender.SendMessage("IP " + args[0] + " unbanned")
	} else {
		sender.SendMessage("IP " + args[0] + " is not banned")
	}
}
