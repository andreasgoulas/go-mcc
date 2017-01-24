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
	"sync"
	"time"
)

type LevelStorage interface {
	Load(path string) (*Level, error)
	Save(level *Level) error
}

type Level struct {
	Name                 string
	Width, Height, Depth uint
	Blocks               []BlockID
	Spawn                Location

	Entities     []Entity
	EntitiesLock sync.RWMutex

	Players     []*Player
	PlayersLock sync.RWMutex
}

func NewLevel(name string, width, height, depth uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		Name:  name,
		Width: width, Height: height, Depth: depth,
		Blocks: make([]BlockID, width*height*depth),
		Spawn: Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(depth) / 2,
		},
		Entities: []Entity{},
		Players:  []*Player{},
	}
}

func (level *Level) Volume() uint {
	return level.Width * level.Height * level.Depth
}

func (level *Level) GetBlock(x, y, z uint) BlockID {
	if x < level.Width && y < level.Height && z < level.Depth {
		return level.Blocks[x+level.Width*(z+level.Depth*y)]
	}

	return BlockAir
}

func (level *Level) SetBlock(x, y, z uint, block BlockID, broadcast bool) {
	if x < level.Width && y < level.Height && z < level.Depth {
		level.Blocks[x+level.Width*(z+level.Depth*y)] = block
		if broadcast {
			level.PlayersLock.RLock()
			for _, player := range level.Players {
				player.SendBlockChange(x, y, z, block)
			}
			level.PlayersLock.RUnlock()
		}
	}
}

func (level *Level) GenerateID() byte {
	for id := byte(0); id < 0xff; id++ {
		free := true
		for _, entity := range level.Entities {
			if entity.GetID() == id {
				free = false
				break
			}
		}

		if free {
			return id
		}
	}

	return 0xff
}

func (level *Level) BroadcastMessage(message string) {
	level.PlayersLock.RLock()
	for _, player := range level.Players {
		player.SendMessage(message)
	}
	level.PlayersLock.RUnlock()
}

func (level *Level) Update(dt time.Duration) {
	level.EntitiesLock.RLock()
	for _, entity := range level.Entities {
		entity.Update(dt)
	}
	level.EntitiesLock.RUnlock()
}

func (level *Level) AddEntity(entity Entity) bool {
	level.EntitiesLock.Lock()
	defer level.EntitiesLock.Unlock()

	id := level.GenerateID()
	if id == 0xff {
		return false
	}

	entity.SetID(id)
	level.Entities = append(level.Entities, entity)

	level.PlayersLock.RLock()
	for _, player := range level.Players {
		player.SendSpawn(entity)
	}
	level.PlayersLock.RUnlock()
	return true
}

func (level *Level) RemoveEntity(entity Entity) {
	level.EntitiesLock.Lock()
	defer level.EntitiesLock.Unlock()

	index := -1
	for i, e := range level.Entities {
		if e == entity {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	level.Entities[index] = level.Entities[len(level.Entities)-1]
	level.Entities[len(level.Entities)-1] = nil
	level.Entities = level.Entities[:len(level.Entities)-1]

	level.PlayersLock.RLock()
	for _, player := range level.Players {
		player.SendDespawn(entity)
	}
	level.PlayersLock.RUnlock()
}

func (level *Level) AddPlayer(player *Player) bool {
	level.PlayersLock.Lock()
	defer level.PlayersLock.Unlock()

	level.EntitiesLock.Lock()
	defer level.EntitiesLock.Unlock()

	id := level.GenerateID()
	if id == 0xff {
		return false
	}

	player.SetID(id)
	level.Entities = append(level.Entities, player)
	for _, p := range level.Players {
		p.SendSpawn(player)
	}

	level.Players = append(level.Players, player)
	player.SendLevel(level)

	for _, entity := range level.Entities {
		player.SendSpawn(entity)
	}
	return true
}

func (level *Level) RemovePlayer(player *Player) {
	level.RemoveEntity(player)

	if player.LoggedIn == 1 {
		level.EntitiesLock.RLock()
		for _, entity := range level.Entities {
			player.SendDespawn(entity)
		}
		level.EntitiesLock.RUnlock()
	}

	level.PlayersLock.Lock()
	defer level.PlayersLock.Unlock()

	index := -1
	for i, p := range level.Players {
		if p == player {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	level.Players[index] = level.Players[len(level.Players)-1]
	level.Players[len(level.Players)-1] = nil
	level.Players = level.Players[:len(level.Players)-1]
}

func (level *Level) FindPlayer(name string) *Player {
	level.PlayersLock.RLock()
	defer level.PlayersLock.RUnlock()

	for _, player := range level.Players {
		if player.Name == name {
			return player
		}
	}

	return nil
}
