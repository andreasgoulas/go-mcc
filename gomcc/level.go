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
	"image/color"
)

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
	Weather     uint
	TexturePack string

	SideBlock       byte
	EdgeBlock       byte
	EdgeHeight      uint
	CloudHeight     uint
	MaxViewDistance uint
	CloudSpeed      float64
	WeatherSpeed    float64
	WeatherFade     float64
	ExpFog          bool
	SideOffset      int

	SkyColor     color.RGBA
	CloudColor   color.RGBA
	FogColor     color.RGBA
	AmbientColor color.RGBA
	DiffuseColor color.RGBA
}

var DefaultColor = color.RGBA{0, 0, 0, 0}

const (
	EnvPropWeather     = 1 << 0
	EnvPropTexturePack = 1 << 1

	EnvPropSideBlock       = 1 << 2
	EnvPropEdgeBlock       = 1 << 3
	EnvPropEdgeHeight      = 1 << 4
	EnvPropCloudHeight     = 1 << 5
	EnvPropMaxViewDistance = 1 << 6
	EnvPropCloudSpeed      = 1 << 7
	EnvPropWeatherSpeed    = 1 << 8
	EnvPropWeatherFade     = 1 << 9
	EnvPropExpFog          = 1 << 10
	EnvPropSideOffset      = 1 << 11

	EnvPropSkyColor     = 1 << 12
	EnvPropCloudColor   = 1 << 13
	EnvPropFogColor     = 1 << 14
	EnvPropAmbientColor = 1 << 15
	EnvPropDiffuseColor = 1 << 16

	EnvPropAll = (EnvPropDiffuseColor << 1) - 1
)

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

	dirty  bool
	width  uint
	height uint
	length uint
	Blocks []byte

	BlockDefs []*BlockDefinition
	Inventory []byte

	MOTD  string
	Spawn Location

	Weather     WeatherType
	TexturePack string
	EnvConfig   EnvConfig
	HackConfig  HackConfig
}

func NewLevel(name string, width, height, length uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		name:   name,
		dirty:  false,
		width:  width,
		height: height,
		length: length,
		Blocks: make([]byte, width*height*length),
		Spawn: Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(length) / 2,
		},
		EnvConfig: EnvConfig{
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
			SkyColor:        DefaultColor,
			CloudColor:      DefaultColor,
			FogColor:        DefaultColor,
			AmbientColor:    DefaultColor,
			DiffuseColor:    DefaultColor,
			Weather:         WeatherSunny,
		},
		HackConfig: HackConfig{
			Flying:          false,
			NoClip:          false,
			Speeding:        false,
			SpawnControl:    true,
			ThirdPersonView: true,
			JumpHeight:      -1,
		},
	}
}

func (level Level) Clone(name string) *Level {
	if len(name) == 0 {
		return nil
	}

	newLevel := level
	newLevel.server = nil
	newLevel.name = name
	newLevel.dirty = true
	newLevel.Blocks = make([]byte, len(level.Blocks))
	copy(newLevel.Blocks, level.Blocks)
	return &newLevel
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

func (level *Level) Position(index uint) (x, y, z uint) {
	x = index % level.width
	y = (index / level.width) / level.length
	z = (index / level.width) % level.length
	return
}

func (level *Level) GetBlock(x, y, z uint) byte {
	if x < level.width && y < level.height && z < level.length {
		return level.Blocks[level.Index(x, y, z)]
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

func (level *Level) ForEachPlayer(fn func(*Player)) {
	if level.server == nil {
		return
	}

	level.server.ForEachPlayer(func(player *Player) {
		if player.level == level {
			fn(player)
		}
	})
}

func (level *Level) SetBlock(x, y, z uint, block byte, broadcast bool) {
	if x < level.width && y < level.height && z < level.length {
		level.dirty = true
		level.Blocks[level.Index(x, y, z)] = block
		if broadcast {
			level.ForEachPlayer(func(player *Player) {
				player.sendBlockChange(x, y, z, block)
			})
		}
	}
}

func (level *Level) SendBlockDefinitions() {
	level.ForEachPlayer(func(player *Player) {
		player.sendBlockDefinitions(level)
	})
}

func (level *Level) SendInventory() {
	level.ForEachPlayer(func(player *Player) {
		player.sendInventory(level)
	})
}

func (level *Level) SendEnvConfig(mask uint32) {
	level.ForEachPlayer(func(player *Player) {
		player.sendEnvConfig(level, mask)
	})
}

func (level *Level) SendHackConfig() {
	level.ForEachPlayer(func(player *Player) {
		player.sendHackConfig(level)
	})
}

func (level *Level) SendMOTD() {
	level.ForEachPlayer(func(player *Player) {
		if player.cpe[CpeInstantMOTD] {
			player.sendMOTD(level)
		} else {
			player.level = nil
			player.despawnLevel(level)
			player.spawnLevel(level)
			player.level = level
		}
	})
}

type BlockBuffer struct {
	level   *Level
	count   uint
	indices [256]int32
	blocks  [256]byte
}

func MakeBlockBuffer(level *Level) BlockBuffer {
	return BlockBuffer{level: level}
}

func (buffer *BlockBuffer) Set(x, y, z uint, block byte) {
	buffer.indices[buffer.count] = int32(buffer.level.Index(x, y, z))
	buffer.blocks[buffer.count] = block
	buffer.count++
	if buffer.count >= 256 {
		buffer.Flush()
	}
}

func (buffer *BlockBuffer) Flush() {
	for i := uint(0); i < buffer.count; i++ {
		index := buffer.indices[i]
		buffer.level.Blocks[index] = buffer.blocks[i]
	}

	buffer.level.dirty = true
	buffer.level.ForEachPlayer(func(player *Player) {
		var blocks [256]byte
		for i := uint(0); i < buffer.count; i++ {
			blocks[i] = player.convertBlock(buffer.blocks[i], buffer.level)
		}

		var packet Packet
		if player.cpe[CpeBulkBlockUpdate] {
			packet.bulkBlockUpdate(buffer.indices[:], blocks[:buffer.count])
		} else {
			for i := uint(0); i < buffer.count; i++ {
				x, y, z := buffer.level.Position(uint(buffer.indices[i]))
				packet.setBlock(x, y, z, blocks[i])
			}
		}

		player.sendPacket(packet)
	})

	buffer.count = 0
}
