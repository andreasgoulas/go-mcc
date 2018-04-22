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
	server.RegisterCommand(&CommandGoto)
	server.RegisterCommand(&CommandKick)
	server.RegisterCommand(&CommandLoad)
	server.RegisterCommand(&CommandMain)
	server.RegisterCommand(&CommandMe)
	server.RegisterCommand(&CommandSave)
	server.RegisterCommand(&CommandSay)
	server.RegisterCommand(&CommandSpawn)
	server.RegisterCommand(&CommandTell)
	server.RegisterCommand(&CommandTp)
	server.RegisterCommand(&CommandUnload)
}
