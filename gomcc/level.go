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

type EnvConfig struct {
	SideBlock       BlockID
	EdgeBlock       BlockID
	EdgeHeight      uint
	CloudHeight     uint
	MaxViewDistance uint
	CloudSpeed      float64
	WeatherSpeed    float64
	WeatherFade     float64
	ExpFog          bool
	SideOffset      int
}

type HackConfig struct {
	Flying          bool
	NoClip          bool
	Speeding        bool
	SpawnControl    bool
	ThirdPersonView bool
	JumpHeight      int
}

type Level struct {
	server *Server
	name   string

	MOTD  string
	Spawn Location

	width, height, length uint
	blocks                []BlockID

	weather     WeatherType
	texturePack string
	envConfig   EnvConfig
	hackConfig  HackConfig
}

func NewLevel(name string, width, height, length uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		name: name,
		Spawn: Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(length) / 2,
		},
		width:   width,
		height:  height,
		length:  length,
		blocks:  make([]BlockID, width*height*length),
		weather: WeatherSunny,
		envConfig: EnvConfig{
			SideBlock:       BlockBedrock,
			EdgeBlock:       BlockActiveWater,
			EdgeHeight:      height / 2,
			CloudHeight:     height + 2,
			MaxViewDistance: 0,
			CloudSpeed:      1.0,
			WeatherSpeed:    1.0,
			WeatherFade:     1.0,
			ExpFog:          false,
			SideOffset:      -2,
		},
		hackConfig: HackConfig{
			Flying:          false,
			NoClip:          false,
			Speeding:        false,
			SpawnControl:    true,
			ThirdPersonView: true,
			JumpHeight:      -1,
		},
	}
}

func (level *Level) Clone(name string) *Level {
	if len(name) == 0 {
		return nil
	}

	blocks := make([]BlockID, len(level.blocks))
	copy(blocks, level.blocks)

	return &Level{
		name:      name,
		Spawn:     level.Spawn,
		width:     level.width,
		height:    level.height,
		length:    level.length,
		blocks:    blocks,
		weather:   level.weather,
		envConfig: level.envConfig,
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

func (level *Level) TexturePack() string {
	return level.texturePack
}

func (level *Level) SetTexturePack(texturePack string) {
	if texturePack == level.texturePack {
		return
	}

	level.texturePack = texturePack
	level.ForEachClient(func(client *Client) {
		client.sendTexturePack(texturePack)
	})
}

func (level *Level) EnvConfig() EnvConfig {
	return level.envConfig
}

func (level *Level) SetEnvConfig(envConfig EnvConfig) {
	if envConfig == level.envConfig {
		return
	}

	level.envConfig = envConfig
	level.ForEachClient(func(client *Client) {
		client.sendEnvConfig(envConfig)
	})
}

func (level *Level) HackConfig() HackConfig {
	return level.hackConfig
}

func (level *Level) SetHackConfig(hackConfig HackConfig) {
	if hackConfig == level.hackConfig {
		return
	}

	level.hackConfig = hackConfig
	level.ForEachClient(func(client *Client) {
		client.sendHackConfig(hackConfig)
	})
}

func (level *Level) SetMOTD(motd string) {
	level.MOTD = motd
	level.ForEachClient(func(client *Client) {
		if client.cpe[CpeInstantMOTD] {
			client.sendMOTD(level)
		} else {
			client.reload()
		}
	})
}
