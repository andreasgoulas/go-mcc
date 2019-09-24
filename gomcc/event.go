// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

type EventType int

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
	EventTypePlayerLogin = iota
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
	EventTypeCommand
)

// EventHandler is the type of the function called to handle an event.
type EventHandler func(eventType EventType, event interface{})

// EventPlayerLogin is dispatched when a player attempts to log in.
// If the event is cancelled, the player will be kicked.
type EventPlayerLogin struct {
	Player       *Player
	Cancel       bool
	CancelReason string
}

// EventPlayerJoin is dispatched when a player joins the server.
type EventPlayerJoin struct {
	Player *Player
}

// EventPlayerQuit is dispatched when a player leaves the server.
type EventPlayerQuit struct {
	Player *Player
}

// EventPlayerChat is dispatched when a player sends a message.
// If the event is cancelled, the message will not be sent.
type EventPlayerChat struct {
	Player  *Player
	Targets []*Player
	Message string
	Format  string
	Cancel  bool
}

// EventPlayerClick is dispatched when a player makes a mouse click.
// If the player is currently targeting another entity, Target will be set.
// If the player is currently targeting a block, BlockX, BlockY, BlockZ,
// BlockFace will be set.
type EventPlayerClick struct {
	Player                 *Player
	Button, Action         byte
	Yaw, Pitch             float64
	Target                 *Entity
	BlockX, BlockY, BlockZ int
	BlockFace              byte
}

// EventEntityLevelChange is dispatched when an entity changes level.
type EventEntityLevelChange struct {
	Entity   *Entity
	From, To *Level
}

// EventEntityMove is dispatched when an entity moves.
// If the event is cancelled, the entity will not move.
type EventEntityMove struct {
	Entity   *Entity
	From, To Location
	Cancel   bool
}

// EventBlockPlace is dispatched when a player places a block.
// If the event is cancelled, the block will not be placed.
type EventBlockPlace struct {
	Player  *Player
	Level   *Level
	Block   byte
	X, Y, Z int
	Cancel  bool
}

// EventBlockBreak is dispatched when a player breaks a block.
// If the event is cancelled, the block will not be broken.
type EventBlockBreak struct {
	Player  *Player
	Level   *Level
	Block   byte
	X, Y, Z int
	Cancel  bool
}

// EventLevelLoad is dispatched when a level is loaded.
type EventLevelLoad struct {
	Level *Level
}

// EventLevelUnload is dispatched when a level is unloaded.
type EventLevelUnload struct {
	Level *Level
}

// EventLevelSave is dispatched when a level is saved.
type EventLevelSave struct {
	Level *Level
}

// EventCommand is dispatched before a command is executed.
// If the event is cancelled, the command will not be executed.
type EventCommand struct {
	Sender  CommandSender
	Command *Command
	Message string
	Cancel  bool
}
