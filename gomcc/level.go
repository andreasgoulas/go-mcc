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

type LevelStorage interface {
	Load(path string) (*Level, error)
	Save(level *Level) error
}

type WeatherType byte

const (
	WeatherSunny   = 0
	WeatherRaining = 1
	WeatherSnowing = 2
)

type LevelAppearance struct {
	TexturePackURL        string
	SideBlock, EdgeBlock  BlockID
	SideLevel, CloudLevel uint
	MaxViewDistance       uint
}

type Level struct {
	server *Server
	name   string

	Spawn Location

	width, height, length uint
	blocks                []BlockID

	appearance LevelAppearance
	weather    WeatherType
}

func NewLevel(name string, width, height, length uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		nil,
		name,
		Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(length) / 2,
		},
		width, height, length,
		make([]BlockID, width*height*length),
		LevelAppearance{
			SideBlock:       BlockBedrock,
			EdgeBlock:       BlockActiveWater,
			SideLevel:       height / 2,
			CloudLevel:      height + 2,
			MaxViewDistance: 0,
		},
		WeatherSunny,
	}
}

func (level *Level) Clone(name string) *Level {
	if len(name) == 0 {
		return nil
	}

	blocks := make([]BlockID, len(level.blocks))
	copy(blocks, level.blocks)

	return &Level{
		nil,
		name,
		level.Spawn,
		level.width, level.height, level.length,
		blocks,
		level.appearance,
		level.weather,
	}
}

func (level *Level) Server() *Server {
	return level.server
}

func (level *Level) Name() string {
	return level.name
}

func (level *Level) Width() uint {
	return level.width
}

func (level *Level) Height() uint {
	return level.height
}

func (level *Level) Length() uint {
	return level.length
}

func (level *Level) Volume() uint {
	return level.width * level.height * level.length
}

func (level *Level) Index(x, y, z uint) uint {
	return x + level.width*(z+level.length*y)
}

func (level *Level) GetBlock(x, y, z uint) BlockID {
	if x < level.width && y < level.height && z < level.length {
		return level.blocks[level.Index(x, y, z)]
	}

	return BlockAir
}

func (level *Level) ForEachEntity(fn func(*Entity)) {
	if level.server == nil {
		return
	}

	level.server.ForEachEntity(func(entity *Entity) {
		if entity.level == level {
			fn(entity)
		}
	})
}

func (level *Level) ForEachClient(fn func(*Client)) {
	if level.server == nil {
		return
	}

	level.server.ForEachClient(func(client *Client) {
		if client.entity.level == level {
			fn(client)
		}
	})
}

func (level *Level) SetBlock(x, y, z uint, block BlockID, broadcast bool) {
	if x < level.width && y < level.height && z < level.length {
		level.blocks[level.Index(x, y, z)] = block
		if broadcast {
			level.ForEachClient(func(client *Client) {
				client.sendBlockChange(x, y, z, block)
			})
		}
	}
}

func (level *Level) Weather() WeatherType {
	return level.weather
}

func (level *Level) SetWeather(weather WeatherType) {
	if weather == level.weather {
		return
	}

	level.weather = weather
	level.ForEachClient(func(client *Client) {
		client.sendWeather(weather)
	})
}

func (level *Level) Appearance() LevelAppearance {
	return level.appearance
}

func (level *Level) SetAppearance(appearance LevelAppearance) {
	if appearance == level.appearance {
		return
	}

	level.appearance = appearance
	level.ForEachClient(func(client *Client) {
		client.sendLevelAppearance(level.appearance)
	})
}
