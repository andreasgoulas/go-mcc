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

package gomcc

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

// ConvertColors returns the given string with each occurence of a %-prefixed
// color code replaced by a client-compatible one.
func ConvertColors(message string) string {
	result := make([]byte, len(message))
	for i := range message {
		result[i] = message[i]
		if message[i] == '%' && i < len(message)-1 {
			color := message[i+1]
			if (color >= 'a' && color <= 'f') ||
				(color >= 'A' && color <= 'A') ||
				(color >= '0' && color <= '9') {
				result[i] = '&'
			}
		}
	}

	return string(result)
}

// A CommandSender is a generic entity that can execute commands and receive
// messages.
type CommandSender interface {
	Name() string
	SendMessage(message string)
	IsOperator() bool
	HasPermission(permission string) bool
}

type CommandHandler func(sender CommandSender, command *Command, message string)

type Command struct {
	Name        string
	Description string
	Permission  string
	Handler     CommandHandler
}
