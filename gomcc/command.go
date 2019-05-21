// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

import (
	"image/color"
	"strings"
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
	Key           uint
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
	HasPermission(permission string) bool
}

// CommandHandler is the type of the function called to execute a command. The
// sender argument is the entity that invoked the command. The message argument
// contains the arguments of the command.
type CommandHandler func(sender CommandSender, command *Command, message string)

// Command describes a command.
type Command struct {
	Name        string
	Description string
	Permission  string
	Handler     CommandHandler
}

// PermissionGroup is a container that holds permission nodes.
type PermissionGroup struct {
	permissions [][]string
}

func checkPermission(permission []string, template []string) bool {
	lenP := len(permission)
	lenT := len(template)
	for i := 0; i < min(lenP, lenT); i++ {
		if template[i] == "*" {
			return true
		} else if permission[i] != template[i] {
			return false
		}
	}

	return lenP == lenT
}

// Clear resets the group to be empty.
func (group *PermissionGroup) Clear() {
	group.permissions = nil
}

// AddPermission adds permission to the group.
func (group *PermissionGroup) AddPermission(permission string) {
	split := strings.Split(permission, ".")
	group.permissions = append(group.permissions, split)
}

// HasPermission reports whether the group contains permission.
func (group *PermissionGroup) HasPermission(permission string) bool {
	if len(permission) == 0 {
		return true
	}

	split := strings.Split(permission, ".")
	for _, template := range group.permissions {
		if checkPermission(split, template) {
			return true
		}
	}

	return false
}
