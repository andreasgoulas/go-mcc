// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"database/sql"
	"net"
	"strings"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *CorePlugin) handleBan(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player> <reason>")
		return
	}

	reason := "You have been banned"
	args := strings.SplitN(message, " ", 2)
	if len(args) > 1 {
		reason = args[1]
	}

	if !gomcc.IsValidName(args[0]) {
		sender.SendMessage(args[0] + " is not a valid name")
		return
	}

	plugin.db.MustExec(`INSERT OR REPLACE INTO banned_names(name, reason, banned_by, timestamp)
VALUES(?, ?, ?, CURRENT_TIMESTAMP)`, args[0], reason, sender.Name())

	sender.SendMessage("Player " + args[0] + " banned")
	if player := sender.Server().FindPlayer(args[0]); player != nil {
		player.Kick(reason)
	}
}

func (plugin *CorePlugin) handleBanIp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <ip> <reason>")
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

	plugin.db.MustExec(`INSERT OR REPLACE INTO banned_ips(ip, reason, banned_by, timestamp)
VALUES(?, ?, ?, CURRENT_TIMESTAMP)`, args[0], reason, sender.Name())

	sender.SendMessage("IP " + args[0] + " banned")
	sender.Server().ForEachPlayer(func(player *gomcc.Player) {
		if player.RemoteAddr() == args[0] {
			player.Kick(reason)
		}
	})
}

func (plugin *CorePlugin) handleKick(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player> <reason>")
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

func (plugin *CorePlugin) handleRank(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		var rank sql.NullString
		if plugin.db.Get(&rank, "SELECT rank FROM players WHERE name = ?", args[0]) == nil {
			if !rank.Valid {
				rank.String = "<nil>"
			}

			sender.SendMessage("The rank of " + args[0] + " is " + rank.String)
		} else {
			sender.SendMessage("Player " + args[0] + " not found")
		}

	case 2:
		var exists bool
		if plugin.db.Get(&exists, "SELECT 1 FROM ranks WHERE name = ?", args[1]) != nil {
			sender.SendMessage("Rank " + args[1] + " not found")
			return
		}

		r := plugin.db.MustExec("UPDATE players SET rank = ? WHERE name = ?", args[1], args[0])
		if num, _ := r.RowsAffected(); num == 0 {
			sender.SendMessage("Player " + args[0] + " not found")
		} else {
			sender.SendMessage("Rank of " + args[0] + " set to " + args[1])
			if player := plugin.FindPlayer(args[0]); player != nil {
				plugin.updatePermissions(player)
			}
		}

	default:
		sender.SendMessage("Usage: " + command.Name + " <player> <rank>")
	}
}

func (plugin *CorePlugin) handleUnban(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	r := plugin.db.MustExec("DELETE FROM banned_names WHERE name = ?", args[0])
	if num, _ := r.RowsAffected(); num == 0 {
		sender.SendMessage("Player " + args[0] + " is not banned")
	} else {
		sender.SendMessage("Player " + args[0] + " unbanned")
	}
}

func (plugin *CorePlugin) handleUnbanIp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <ip>")
		return
	}

	r := plugin.db.MustExec("DELETE FROM banned_ips WHERE ip = ?", args[0])
	if num, _ := r.RowsAffected(); num == 0 {
		sender.SendMessage("IP " + args[0] + " is not banned")
	} else {
		sender.SendMessage("IP " + args[0] + " unbanned")
	}
}
