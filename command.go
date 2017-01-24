// Copyright 2017 Andrew Goulas
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

package main

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

type CommandSender interface {
	SendMessage(message string)
	IsOperator() bool
}

type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
}

type CommandHandler interface {
	HandleCommand(sender CommandSender, command *Command, args []string)
}
