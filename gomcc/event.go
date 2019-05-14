// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

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
	Player       *Player
	Cancel       bool
	CancelReason string
}

type EventPlayerLogin struct {
	Player       *Player
	Cancel       bool
	CancelReason string
}

type EventPlayerJoin struct {
	Player *Player
}

type EventPlayerQuit struct {
	Player *Player
}

type EventPlayerChat struct {
	Player  *Player
	Targets []*Player
	Message string
	Format  string
	Cancel  bool
}

type EventPlayerClick struct {
	Player                 *Player
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
	Player  *Player
	Level   *Level
	Block   byte
	X, Y, Z uint
	Cancel  bool
}

type EventBlockBreak struct {
	Player  *Player
	Level   *Level
	Block   byte
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
