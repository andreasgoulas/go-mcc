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

import (
	"math"
)

// A Location represents the location of an entity in a world.
// Yaw and Pitch are specified in degrees.
type Location struct {
	X, Y, Z, Yaw, Pitch float64
}

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
)

type Entity struct {
	Client *Client
	Server *Server

	NameID byte

	Name        string
	DisplayName string
	ListName    string

	GroupName string
	GroupRank byte

	modelName string
	skinName  string

	level        *Level
	location     Location
	lastLocation Location
}

func NewEntity(name string, server *Server) *Entity {
	return &Entity{
		Server:      server,
		NameID:      0xff,
		Name:        name,
		DisplayName: name,
		ListName:    name,
		modelName:   ModelHumanoid,
		skinName:    name,
	}
}

func (entity *Entity) Location() Location {
	return entity.location
}

func (entity *Entity) Teleport(location Location) {
	if location == entity.location {
		return
	}

	event := &EventEntityMove{entity, entity.location, location, false}
	entity.Server.FireEvent(EventTypeEntityMove, &event)
	if event.Cancel {
		return
	}

	entity.location = location
	if entity.Client != nil {
		entity.Client.sendTeleport(entity)
	}
}

func (entity *Entity) Level() *Level {
	return entity.level
}

func (entity *Entity) TeleportLevel(level *Level) {
	if entity.level == level {
		return
	}

	lastLevel := entity.level
	if entity.level != nil {
		entity.level = nil
		entity.despawn(lastLevel)
	}

	if level != nil {
		entity.location = level.Spawn
		entity.lastLocation = Location{}
		entity.spawn(level)
	}

	entity.level = level

	event := EventEntityLevelChange{entity, lastLevel, level}
	entity.Server.FireEvent(EventTypeEntityLevelChange, &event)
}

func (entity *Entity) Model() string {
	return entity.modelName
}

func (entity *Entity) SetModel(modelName string) {
	if modelName == entity.modelName {
		return
	}

	entity.modelName = modelName
	if entity.level != nil {
		entity.level.ForEachClient(func(client *Client) {
			client.sendChangeModel(entity)
		})
	}
}

func (entity *Entity) Skin() string {
	return entity.skinName
}

func (entity *Entity) SetSkin(skinName string) {
	if skinName == entity.skinName {
		return
	}

	entity.skinName = skinName
	if entity.level != nil {
		entity.level.ForEachClient(func(client *Client) {
			client.sendSpawn(entity)
		})
	}
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

	var packet interface{}
	if teleport {
		packet = &packetPlayerTeleport{
			packetTypePlayerTeleport,
			entity.NameID,
			int16(entity.location.X * 32),
			int16(entity.location.Y * 32),
			int16(entity.location.Z * 32),
			byte(entity.location.Yaw * 256 / 360),
			byte(entity.location.Pitch * 256 / 360),
		}
	} else if positionDirty && rotationDirty {
		packet = &packetPositionOrientationUpdate{
			packetTypePositionOrientationUpdate,
			entity.NameID,
			byte((entity.location.X - entity.lastLocation.X) * 32),
			byte((entity.location.Y - entity.lastLocation.Y) * 32),
			byte((entity.location.Z - entity.lastLocation.Z) * 32),
			byte(entity.location.Yaw * 256 / 360),
			byte(entity.location.Pitch * 256 / 360),
		}
	} else if positionDirty {
		packet = &packetPositionUpdate{
			packetTypePositionUpdate,
			entity.NameID,
			byte((entity.location.X - entity.lastLocation.X) * 32),
			byte((entity.location.Y - entity.lastLocation.Y) * 32),
			byte((entity.location.Z - entity.lastLocation.Z) * 32),
		}
	} else if rotationDirty {
		packet = &packetOrientationUpdate{
			packetTypeOrientationUpdate,
			entity.NameID,
			byte(entity.location.Yaw * 256 / 360),
			byte(entity.location.Pitch * 256 / 360),
		}
	} else {
		return
	}

	entity.lastLocation = entity.location
	entity.level.ForEachClient(func(client *Client) {
		if client != entity.Client {
			client.sendPacket(packet)
		}
	})
}

func (entity *Entity) spawn(level *Level) {
	level.ForEachClient(func(client *Client) {
		client.sendSpawn(entity)
	})

	if entity.Client != nil {
		entity.Client.sendLevel(level)
		entity.Client.sendSpawn(entity)
		level.ForEachEntity(func(other *Entity) {
			entity.Client.sendSpawn(other)
		})
	}
}

func (entity *Entity) despawn(level *Level) {
	level.ForEachClient(func(client *Client) {
		client.sendDespawn(entity)
	})

	if entity.Client != nil && entity.Client.loggedIn == 1 {
		entity.Client.sendDespawn(entity)
		level.ForEachEntity(func(other *Entity) {
			entity.Client.sendDespawn(other)
		})
	}
}
