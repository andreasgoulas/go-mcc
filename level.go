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

type LevelStorage interface {
	Load(path string) (*Level, error)
	Save(level *Level) error
}

type LevelAppearance struct {
	TexturePackURL        string
	SideBlock, EdgeBlock  BlockID
	SideLevel, CloudLevel uint
	MaxViewDistance       uint
}

type Level struct {
	Name                 string
	Width, Height, Depth uint
	Blocks               []BlockID
	Spawn                Location
	Appearance           LevelAppearance
}

func NewLevel(name string, width, height, depth uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		name,
		width, height, depth,
		make([]BlockID, width*height*depth),
		Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(depth) / 2,
		},
		LevelAppearance{
			SideBlock:       BlockBedrock,
			EdgeBlock:       BlockActiveWater,
			SideLevel:       height / 2,
			CloudLevel:      height + 2,
			MaxViewDistance: 0,
		},
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

func (level *Level) SetBlock(x, y, z uint, block BlockID, server *Server) {
	if x < level.Width && y < level.Height && z < level.Depth {
		level.Blocks[x+level.Width*(z+level.Depth*y)] = block
		if server != nil {
			server.ClientsLock.RLock()
			for _, client := range server.Clients {
				if client.Entity.Level == level {
					client.SendBlockChange(x, y, z, block)
				}
			}
			server.ClientsLock.RUnlock()
		}
	}
}
