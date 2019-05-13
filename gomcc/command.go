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

type HotKeyDesc struct {
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

// A Command describes a command.
type Command struct {
	Name        string
	Description string
	Permission  string
	Handler     CommandHandler
}

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

func (group *PermissionGroup) Clear() {
	group.permissions = nil
}

func (group *PermissionGroup) AddPermission(permission string) {
	split := strings.Split(permission, ".")
	group.permissions = append(group.permissions, split)
}

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
