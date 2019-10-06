// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package mcc

import (
	"math"
)

const (
	ModelChicken   = "chicken"
	ModelCreeper   = "creeper"
	ModelCrocodile = "croc"
	ModelHumanoid  = "humanoid"
	ModelPig       = "pig"
	ModelPrinter   = "printer"
	ModelSheep     = "sheep"
	ModelSkeleton  = "skeleton"
	ModelSpider    = "spider"
	ModelZombie    = "zombie"
	ModelHead      = "head"
	ModelSitting   = "sitting"
	ModelChibi     = "chibi"
)

// EntityProps holds various entity properties.
type EntityProps struct {
	RotX, RotY, RotZ       float64
	ScaleX, ScaleY, ScaleZ float64
}

const (
	EntityPropRotX   = 1 << 0
	EntityPropRotY   = 1 << 1
	EntityPropRotZ   = 1 << 2
	EntityPropScaleX = 1 << 3
	EntityPropScaleY = 1 << 4
	EntityPropScaleZ = 1 << 5

	EntityPropAll = (EntityPropScaleZ << 1) - 1
)

// Entity represents a base entity.
type Entity struct {
	server *Server
	player *Player

	id   byte
	name string

	Model string
	Props EntityProps

	DisplayName string
	SkinName    string

	ListName  string
	GroupName string
	GroupRank byte

	level        *Level
	location     Location
	lastLocation Location
}

// NewEntity creates a new Entity with the specified name.
func NewEntity(name string, server *Server) *Entity {
	return &Entity{
		server:      server,
		id:          0xff,
		name:        name,
		Model:       ModelHumanoid,
		Props:       EntityProps{ScaleX: 1.0, ScaleY: 1.0, ScaleZ: 1.0},
		DisplayName: name,
		SkinName:    name,
		ListName:    name,
	}
}

func (entity *Entity) Server() *Server {
	return entity.server
}

func (entity *Entity) ID() byte {
	return entity.id
}

func (entity *Entity) Name() string {
	return entity.name
}

// SendModel sends the model of the entity to all relevant players.
func (entity *Entity) SendModel() {
	if entity.level != nil {
		entity.level.ForEachPlayer(func(player *Player) {
			player.sendChangeModel(entity)
		})
	}
}

// SendProps sends the EntityProps of the entity to all relevant players.
// mask controls which properties are sent.
func (entity *Entity) SendProps(mask uint32) {
	if entity.level != nil {
		entity.level.ForEachPlayer(func(player *Player) {
			player.sendEntityProps(entity, mask)
		})
	}
}

// SendListName sends the tab list name of the entity to all relevant players.
func (entity *Entity) SendListName() {
	entity.server.ForEachPlayer(func(player *Player) {
		player.sendAddPlayerList(entity)
	})
}

func (entity *Entity) Location() Location {
	return entity.location
}

// Teleport teleports the entity to location.
func (entity *Entity) Teleport(location Location) {
	if location == entity.location {
		return
	}

	event := &EventEntityMove{entity, entity.location, location, false}
	entity.server.FireEvent(EventTypeEntityMove, &event)
	if event.Cancel {
		return
	}

	entity.location = location
	if entity.player != nil {
		entity.player.sendTeleport(entity)
	}
}

func (entity *Entity) Level() *Level {
	return entity.level
}

// TeleportLevel teleports the entity to the spawn location of level.
func (entity *Entity) TeleportLevel(level *Level) {
	if entity.level == level {
		return
	}

	lastLevel := entity.level
	if lastLevel != nil {
		entity.level = nil
		entity.despawn(lastLevel)
		if entity.player != nil {
			entity.player.despawnLevel(lastLevel)
		}
	}

	if level != nil {
		entity.location = level.Spawn
		entity.lastLocation = entity.location
		if entity.player != nil {
			entity.player.spawnLevel(level)
		}

		entity.spawn(level)
	}

	entity.level = level

	event := EventEntityLevelChange{entity, lastLevel, level}
	entity.server.FireEvent(EventTypeEntityLevelChange, &event)
}

func (entity *Entity) update() {
	if entity.level == nil {
		return
	}

	positionDirty := false
	if entity.location.X != entity.lastLocation.X ||
		entity.location.Y != entity.lastLocation.Y ||
		entity.location.Z != entity.lastLocation.Z {
		positionDirty = true
	}

	rotationDirty := false
	if entity.location.Yaw != entity.lastLocation.Yaw ||
		entity.location.Pitch != entity.lastLocation.Pitch {
		rotationDirty = true
	}

	teleport := false
	if math.Abs(entity.location.X-entity.lastLocation.X) > 1.0 ||
		math.Abs(entity.location.Y-entity.lastLocation.Y) > 1.0 ||
		math.Abs(entity.location.Z-entity.lastLocation.Z) > 1.0 {
		teleport = true
	}

	var packet, packetExt packet
	if teleport {
		packet.teleport(entity, false, false)
		packetExt.teleport(entity, false, true)
	} else if positionDirty && rotationDirty {
		packet.positionOrientationUpdate(entity)
	} else if positionDirty {
		packet.positionUpdate(entity)
	} else if rotationDirty {
		packet.orientationUpdate(entity)
	} else {
		return
	}

	entity.lastLocation = entity.location
	entity.level.ForEachPlayer(func(player *Player) {
		if player.Entity != entity {
			if teleport && player.cpe[CpeExtEntityPositions] {
				player.sendPacket(packetExt)
			} else {
				player.sendPacket(packet)
			}
		}
	})
}

// Respawn respawns the entity to all relevant players.
func (entity *Entity) Respawn() {
	if entity.level == nil {
		return
	}

	entity.despawn(entity.level)
	entity.location = entity.level.Spawn
	entity.lastLocation = entity.location
	entity.spawn(entity.level)
}

func (entity *Entity) spawn(level *Level) {
	level.ForEachPlayer(func(player *Player) {
		player.sendSpawn(entity)
	})
}

func (entity *Entity) despawn(level *Level) {
	level.ForEachPlayer(func(player *Player) {
		player.sendDespawn(entity)
	})
}
