// Copyright 2017-2018 Andrew Goulas
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

func HandleTell(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.SplitN(message, " ", 2)
	if len(args) < 2 {
		sender.SendMessage("Usage: " + command.Name + " <player> <message>")
		return
	}

	found := sender.Server().FindEntity(args[0], func(entity *gomcc.Entity) {
		if entity.Client == nil {
			return
		}

		message = gomcc.ConvertColors(args[1])
		sender.SendMessage("[me -> " + entity.Client.Name() + "] " + message)
		entity.Client.SendMessage("[" + sender.Name() + " -> me] " + message)
	})

	if !found {
		sender.SendMessage("Player " + args[0] + " not found")
	}
}
