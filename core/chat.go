package main

import (
	"log"
	"sort"
	"strings"

	"github.com/andreasgoulas/go-mcc/mcc"
)

func (plugin *plugin) privateMessage(message string, src, dst mcc.CommandSender) {
	srcNick := src.Name()
	dstNick := dst.Name()

	var srcPlayer *player
	if player, ok := src.(*mcc.Player); ok {
		srcNick = player.Nickname
		srcPlayer = plugin.findPlayer(src.Name())
		if srcPlayer.mute {
			src.SendMessage("You are muted")
			return
		}
	}

	var dstPlayer *player
	if player, ok := dst.(*mcc.Player); ok {
		dstNick = player.Nickname
		dstPlayer = plugin.findPlayer(dst.Name())
	}

	src.SendMessage("to " + dstNick + ": &f" + message)
	if dstPlayer != nil {
		if dstPlayer.isIgnored(src.Name()) {
			return
		} else if srcPlayer != nil {
			dstPlayer.lastSender = src.Name()
		}
	}

	dst.SendMessage("from " + srcNick + ": &f" + message)
}

func (plugin *plugin) broadcastMessage(src mcc.CommandSender, message string) {
	log.Println(message)
	src.Server().ForEachPlayer(func(player *mcc.Player) {
		if !plugin.findPlayer(player.Name()).isIgnored(src.Name()) {
			player.SendMessage(message)
		}
	})
}

func (plugin *plugin) handleIgnore(sender mcc.CommandSender, command *mcc.Command, message string) {
	if _, ok := sender.(*mcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	args := strings.Fields(message)
	switch len(args) {
	case 0:
		player := plugin.findPlayer(sender.Name())
		if len(player.ignoreList) == 0 {
			sender.SendMessage("You are not ignoring anyone")
			return
		}

		players := make([]string, len(player.ignoreList))
		copy(players, player.ignoreList)
		sort.Strings(players)
		sender.SendMessage(strings.Join(players, ", "))

	case 1:
		if !mcc.IsValidName(args[0]) {
			sender.SendMessage(args[0] + " is not a valid name")
			return
		}

		if args[0] == sender.Name() {
			sender.SendMessage("You cannot ignore yourself")
			return
		}

		found := false
		player := plugin.findPlayer(sender.Name())
		for i, name := range player.ignoreList {
			if name == args[0] {
				found = true
				player.ignoreList = append(player.ignoreList[:i], player.ignoreList[i+1:]...)
				sender.SendMessage("You are no longer ignoring " + args[0])
				break
			}
		}

		if !found {
			player.ignoreList = append(player.ignoreList, args[0])
			sender.SendMessage("You are ignoring " + args[0])
		}

	default:
		command.PrintUsage(sender)
	}
}

func (plugin *plugin) handleMe(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) == 0 {
		command.PrintUsage(sender)
		return
	}

	name := sender.Name()
	if player, ok := sender.(*mcc.Player); ok {
		if plugin.findPlayer(name).mute {
			sender.SendMessage("You are muted")
			return
		}

		name = player.Nickname
	}

	plugin.broadcastMessage(sender, "* "+name+" "+message)
}

func (plugin *plugin) handleMute(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	if len(args) != 1 {
		command.PrintUsage(sender)
		return
	}

	if player := plugin.findPlayer(args[0]); player != nil {
		player.mute = !player.mute
		if player.mute {
			sender.SendMessage("Player " + args[0] + " muted")
		} else {
			sender.SendMessage("Player " + args[0] + " unmuted")
		}
	} else {
		sender.SendMessage("Player " + args[0] + " not found")
	}
}

func (plugin *plugin) handleNick(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.Fields(message)
	switch len(args) {
	case 1:
		player := sender.Server().FindPlayer(args[0])
		if player == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		player.Nickname = player.Name()
		sender.SendMessage("Nick of " + args[0] + " reset")

	case 2:
		if !mcc.IsValidName(args[1]) {
			sender.SendMessage(args[1] + " is not a valid name")
			return
		}

		player := sender.Server().FindPlayer(args[0])
		if player == nil {
			sender.SendMessage("Player " + args[0] + " not found")
			return
		}

		player.Nickname = args[1]
		sender.SendMessage("Nick of " + args[0] + " set to " + args[1])

	default:
		command.PrintUsage(sender)
	}
}

func (plugin *plugin) handleR(sender mcc.CommandSender, command *mcc.Command, message string) {
	if _, ok := sender.(*mcc.Player); !ok {
		sender.SendMessage("You are not a player")
		return
	}

	if len(message) == 0 {
		command.PrintUsage(sender)
		return
	}

	player := plugin.findPlayer(sender.Name())
	lastSender := sender.Server().FindPlayer(player.lastSender)
	if lastSender == nil {
		sender.SendMessage("Player not found")
		return
	}

	plugin.privateMessage(message, sender, lastSender)
}

func (plugin *plugin) handleSay(sender mcc.CommandSender, command *mcc.Command, message string) {
	if len(message) == 0 {
		command.PrintUsage(sender)
		return
	}

	sender.Server().BroadcastMessage(message)
}

func (plugin *plugin) handleTell(sender mcc.CommandSender, command *mcc.Command, message string) {
	args := strings.SplitN(message, " ", 2)
	if len(args) < 2 {
		command.PrintUsage(sender)
		return
	}

	player := sender.Server().FindPlayer(args[0])
	if player == nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	plugin.privateMessage(args[1], sender, player)
}
