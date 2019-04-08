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

type EventType uint

const (
	ButtonLeft   = 0
	ButtonRight  = 1
	ButtonMiddle = 2
)

const (
	ButtonPress   = 0
	ButtonRelease = 1
)

const (
	EventTypePlayerPreLogin = iota
	EventTypePlayerLogin
	EventTypePlayerJoin
	EventTypePlayerQuit
	EventTypePlayerChat
	EventTypePlayerClick
	EventTypeEntityLevelChange
	EventTypeEntityMove
	EventTypeBlockPlace
	EventTypeBlockBreak
	EventTypeLevelLoad
	EventTypeLevelUnload
	EventTypeLevelSave
	EventTypeMessage
)

type EventHandler func(eventType EventType, event interface{})

type EventPlayerPreLogin struct {
	Client       *Client
	Cancel       bool
	CancelReason string
}

type EventPlayerLogin struct {
	Client       *Client
	Cancel       bool
	CancelReason string
}

type EventPlayerJoin struct {
	Client *Client
}

type EventPlayerQuit struct {
	Client *Client
}

type EventPlayerChat struct {
	Client  *Client
	Message string
	Cancel  bool
}

type EventPlayerClick struct {
	Client                 *Client
	Button, Action         byte
	Yaw, Pitch             float64
	Target                 *Entity
	BlockX, BlockY, BlockZ uint
	BlockFace              byte
}

type EventEntityLevelChange struct {
	Entity   *Entity
	From, To *Level
}

type EventEntityMove struct {
	Entity   *Entity
	From, To Location
	Cancel   bool
}

type EventBlockPlace struct {
	Entity  *Entity
	Level   *Level
	Block   BlockID
	X, Y, Z uint
	Cancel  bool
}

type EventBlockBreak struct {
	Entity  *Entity
	Level   *Level
	Block   BlockID
	X, Y, Z uint
	Cancel  bool
}

type EventLevelLoad struct {
	Level *Level
}

type EventLevelUnload struct {
	Level *Level
}

type EventLevelSave struct {
	Level *Level
}

type EventMessage struct {
	Sender  *CommandSender
	Message string
	Cancel  bool
}
