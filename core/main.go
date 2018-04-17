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
	"Go-MCC/gomcc"
)

func Initialize(server *gomcc.Server) {
	server.RegisterCommand(&gomcc.Command{
		"goto",
		"Move to another level.",
		"core.goto",
		HandleGoto,
	})

	server.RegisterCommand(&gomcc.Command{
		"me",
		"Broadcast an action.",
		"core.me",
		HandleMe,
	})

	server.RegisterCommand(&gomcc.Command{
		"say",
		"Broadcast a message.",
		"core.say",
		HandleSay,
	})

	server.RegisterCommand(&gomcc.Command{
		"spawn",
		"Teleport to the spawn location of the level.",
		"core.spawn",
		HandleSpawn,
	})

	server.RegisterCommand(&gomcc.Command{
		"tell",
		"Send a private message to a player.",
		"core.tell",
		HandleTell,
	})

	server.RegisterCommand(&gomcc.Command{
		"tp",
		"Teleport to another player.",
		"core.tp",
		HandleTp,
	})
}
