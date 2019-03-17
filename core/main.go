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
	"Go-MCC/gomcc"
)

var lastSender map[string]string

func Initialize(server *gomcc.Server) {
	lastSender = make(map[string]string)

	server.RegisterCommand(&commandCopyLvl)
	server.RegisterCommand(&commandGoto)
	server.RegisterCommand(&commandKick)
	server.RegisterCommand(&commandLoad)
	server.RegisterCommand(&commandMain)
	server.RegisterCommand(&commandMe)
	server.RegisterCommand(&commandNewLvl)
	server.RegisterCommand(&commandNick)
	server.RegisterCommand(&commandR)
	server.RegisterCommand(&commandSave)
	server.RegisterCommand(&commandSay)
	server.RegisterCommand(&commandSetSpawn)
	server.RegisterCommand(&commandSkin)
	server.RegisterCommand(&commandSpawn)
	server.RegisterCommand(&commandSummon)
	server.RegisterCommand(&commandTell)
	server.RegisterCommand(&commandTp)
	server.RegisterCommand(&commandUnload)
}
