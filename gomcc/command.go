// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

import (
	"image/color"
)

const (
	ColorBlack       = "&0"
	ColorDarkBlue    = "&1"
	ColorDarkGreen   = "&2"
	ColorDarkAqua    = "&3"
	ColorDarkRed     = "&4"
	ColorDarkPurple  = "&5"
	ColorGold        = "&6"
	ColorGray        = "&7"
	ColorDarkGray    = "&8"
	ColorBlue        = "&9"
	ColorGreen       = "&a"
	ColorAqua        = "&b"
	ColorRed         = "&c"
	ColorLightPurple = "&d"
	ColorYellow      = "&e"
	ColorWhite       = "&f"

	ColorDefault = ColorWhite
)

// ColorDesc describes a chat color.
type ColorDesc struct {
	color.RGBA
	Code, Fallback byte
}

const (
	KeyModNone  = 0
	KeyModCtrl  = 1
	KeyModShift = 2
	KeyModAlt   = 4
)

// HotKeyDesc describes a text hotkey.
type HotkeyDesc struct {
	Label, Action string
	Key           int
	KeyMods       byte
}

const (
	MessageChat         = 0
	MessageStatus1      = 1
	MessageStatus2      = 2
	MessageStatus3      = 3
	MessageBottomRight1 = 11
	MessageBottomRight2 = 12
	MessageBottomRight3 = 13
	MessageAnnouncement = 100
)

// A CommandSender is a generic entity that can execute commands and receive
// messages.
type CommandSender interface {
	Server() *Server
	Name() string
	SendMessage(message string)
	HasPermission(command *Command) bool
}

// CommandHandler is the type of the function called to execute a command. The
// sender argument is the entity that invoked the command. The message argument
// contains the arguments of the command.
type CommandHandler func(sender CommandSender, command *Command, message string)

// Command describes a command.
type Command struct {
	Name        string
	Description string
	Permissions uint32
	Handler     CommandHandler
}

// Rank represents a group of players that have the same permissions.
type Rank struct {
	Name        string
	Tag         string
	Permissions uint32
	Rules       map[string]bool
	CanPlace    [BlockCount]bool
	CanBreak    [BlockCount]bool
}

// DefaultRank stores the default player permissions.
var DefaultRank = func() (rank Rank) {
	for i := 0; i < BlockCount; i++ {
		rank.CanPlace[i] = true
		rank.CanBreak[i] = true
	}

	banned := []byte{BlockBedrock, BlockActiveWater, BlockWater, BlockActiveLava, BlockLava}
	for _, block := range banned {
		rank.CanPlace[block] = false
	}
	rank.CanBreak[BlockBedrock] = false

	return
}()
