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
	"strings"

	"Go-MCC/gomcc"
)

var commandBan = gomcc.Command{
	Name:        "ban",
	Description: "Ban a player from the server.",
	Permission:  "core.ban",
	Handler:     handleBan,
}

func handleBan(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player> <reason>")
		return
	}

	reason := "You have been banned"
	args := strings.SplitN(message, " ", 2)
	if len(args) > 1 {
		reason = args[1]
	}

	if banned, _ := CoreDb.IsBanned(BanTypeName, args[0]); banned {
		sender.SendMessage("Player " + args[0] + " is already banned")
		return
	}

	CoreDb.Ban(BanTypeName, args[0], reason, sender.Name())

	client := sender.Server().FindClient(args[0])
	if client != nil {
		client.Kick(reason)
	}

	sender.SendMessage("Player " + args[0] + " banned")
}

var commandBanIp = gomcc.Command{
	Name:        "banip",
	Description: "Ban an IP address from the server.",
	Permission:  "core.banip",
	Handler:     handleBanIp,
}

func handleBanIp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <ip> <reason>")
		return
	}

	reason := "You have been banned"
	args := strings.SplitN(message, " ", 2)
	if len(args) > 1 {
		reason = args[1]
	}

	if banned, _ := CoreDb.IsBanned(BanTypeIp, args[0]); banned {
		sender.SendMessage("IP " + args[0] + " is already banned")
		return
	}

	CoreDb.Ban(BanTypeIp, args[0], reason, sender.Name())

	sender.Server().ForEachClient(func(client *gomcc.Client) {
		if client.RemoteAddr() == args[0] {
			client.Kick(reason)
		}
	})

	sender.SendMessage("IP " + args[0] + " banned")
}

var commandKick = gomcc.Command{
	Name:        "kick",
	Description: "Kick a player from the server.",
	Permission:  "core.kick",
	Handler:     handleKick,
}

func handleKick(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	if len(message) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player> <reason>")
		return
	}

	args := strings.SplitN(message, " ", 2)
	player := sender.Server().FindClient(args[0])
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

var commandRank = gomcc.Command{
	Name:        "rank",
	Description: "Set the rank of a player.",
	Permission:  "core.rank",
	Handler:     handleRank,
}

func handleRank(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		rank := CoreDb.Rank(args[0])
		if len(rank) == 0 {
			sender.SendMessage(args[0] + " has no rank assigned")
		} else {
			sender.SendMessage("The rank of " + args[0] + " is " + rank)
		}

	case 2:
		if !CoreDb.RankExists(args[1]) {
			sender.SendMessage("Rank " + args[1] + " does not exist")
			return
		}

		CoreDb.SetRank(args[0], args[1])

		if client := sender.Server().FindClient(args[0]); client != nil {
			client.SetPermissions(CoreDb.PlayerPermissions(args[0]))
		}

	default:
		sender.SendMessage("Usage: " + command.Name + " <player> <rank>")
	}
}

var commandUnban = gomcc.Command{
	Name:        "unban",
	Description: "Remove the ban for a player.",
	Permission:  "core.unban",
	Handler:     handleUnban,
}

func handleUnban(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	CoreDb.Unban(BanTypeName, args[0])
	sender.SendMessage("Player " + args[0] + " unbanned")
}

var commandUnbanIp = gomcc.Command{
	Name:        "unbanip",
	Description: "Remove the ban for an IP address.",
	Permission:  "core.unbanip",
	Handler:     handleUnbanIp,
}

func handleUnbanIp(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		sender.SendMessage("Usage: " + command.Name + " <ip>")
		return
	}

	CoreDb.Unban(BanTypeIp, args[0])
	sender.SendMessage("IP " + args[0] + " unbanned")
}
