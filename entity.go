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

package main

import (
	"math"
	"time"
)

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
	NameID byte

	Name        string
	DisplayName string
	ListName    string

	ModelName string
	SkinName  string

	GroupName string
	GroupRank byte

	Client       *Client
	Server       *Server
	Level        *Level
	Location     Location
	LastLocation Location
}

func NewEntity(name string, server *Server) *Entity {
	return &Entity{
		NameID:      0xff,
		Name:        name,
		DisplayName: name,
		ListName:    name,
		ModelName:   ModelHumanoid,
		SkinName:    name,
		Server:      server,
	}
}

func (entity *Entity) Teleport(location Location) {
	if location == entity.Location {
		return
	}

	event := &EventEntityMove{entity, entity.Location, location, false}
	entity.Server.FireEvent(EventTypeEntityMove, &event)
	if event.Cancel {
		return
	}

	entity.Location = location
}

func (entity *Entity) TeleportLevel(level *Level) {
	if entity.Level == level {
		return
	}

	lastLevel := entity.Level
	if entity.Level != nil {
		entity.Level = nil
		entity.Despawn(lastLevel)
	}

	if level != nil {
		entity.Location = level.Spawn
		entity.LastLocation = Location{}
		entity.Spawn(level)
	}

	entity.Level = level

	event := EventEntityLevelChange{entity, lastLevel, level}
	entity.Server.FireEvent(EventTypeEntityLevelChange, &event)
}

func (entity *Entity) SetModel(modelName string) {
	if modelName == entity.ModelName {
		return
	}

	entity.ModelName = modelName
	if entity.Level != nil {
		entity.Level.ForEachClient(func(client *Client) {
			client.SendChangeModel(entity)
		})
	}
}

func (entity *Entity) Update(dt time.Duration) {
	if entity.Level == nil {
		return
	}

	positionDirty := false
	if entity.Location.X != entity.LastLocation.X ||
		entity.Location.Y != entity.LastLocation.Y ||
		entity.Location.Z != entity.LastLocation.Z {
		positionDirty = true
	}

	rotationDirty := false
	if entity.Location.Yaw != entity.LastLocation.Yaw ||
		entity.Location.Pitch != entity.LastLocation.Pitch {
		rotationDirty = true
	}

	teleport := false
	if math.Abs(entity.Location.X-entity.LastLocation.X) > 1.0 ||
		math.Abs(entity.Location.Y-entity.LastLocation.Y) > 1.0 ||
		math.Abs(entity.Location.Z-entity.LastLocation.Z) > 1.0 {
		teleport = true
	}

	var packet interface{}
	if teleport {
		packet = &PacketPlayerTeleport{
			PacketTypePlayerTeleport,
			entity.NameID,
			int16(entity.Location.X * 32),
			int16(entity.Location.Y * 32),
			int16(entity.Location.Z * 32),
			byte(entity.Location.Yaw * 256 / 360),
			byte(entity.Location.Pitch * 256 / 360),
		}
	} else if positionDirty && rotationDirty {
		packet = &PacketPositionOrientationUpdate{
			PacketTypePositionOrientationUpdate,
			entity.NameID,
			byte((entity.Location.X - entity.LastLocation.X) * 32),
			byte((entity.Location.Y - entity.LastLocation.Y) * 32),
			byte((entity.Location.Z - entity.LastLocation.Z) * 32),
			byte(entity.Location.Yaw * 256 / 360),
			byte(entity.Location.Pitch * 256 / 360),
		}
	} else if positionDirty {
		packet = &PacketPositionUpdate{
			PacketTypePositionUpdate,
			entity.NameID,
			byte((entity.Location.X - entity.LastLocation.X) * 32),
			byte((entity.Location.Y - entity.LastLocation.Y) * 32),
			byte((entity.Location.Z - entity.LastLocation.Z) * 32),
		}
	} else if rotationDirty {
		packet = &PacketOrientationUpdate{
			PacketTypeOrientationUpdate,
			entity.NameID,
			byte(entity.Location.Yaw * 256 / 360),
			byte(entity.Location.Pitch * 256 / 360),
		}
	} else {
		return
	}

	entity.LastLocation = entity.Location
	entity.Level.ForEachClient(func(client *Client) {
		if client != entity.Client {
			client.SendPacket(packet)
		}
	})
}

func (entity *Entity) Spawn(level *Level) {
	level.ForEachClient(func(client *Client) {
		client.SendSpawn(entity)
	})

	if entity.Client != nil {
		entity.Client.SendLevel(level)
		entity.Client.SendSpawn(entity)
		level.ForEachEntity(func(other *Entity) {
			entity.Client.SendSpawn(other)
		})
	}
}

func (entity *Entity) Despawn(level *Level) {
	level.ForEachClient(func(client *Client) {
		client.SendDespawn(entity)
	})

	if entity.Client != nil && entity.Client.LoggedIn == 1 {
		entity.Client.SendDespawn(entity)
		level.ForEachEntity(func(other *Entity) {
			entity.Client.SendDespawn(other)
		})
	}
}
