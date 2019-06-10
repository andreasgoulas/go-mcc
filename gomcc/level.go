// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

import (
	"image/color"
	"time"
)

// LevelStorage is the interface that must be implemented by storage backends
// that can import and export levels.
type LevelStorage interface {
	Load(name string) (*Level, error)
	Save(level *Level) error
}

const (
	WeatherSunny   = 0
	WeatherRaining = 1
	WeatherSnowing = 2
)

// EnvConfig specifies the appearance of a level.
type EnvConfig struct {
	Weather     byte
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

// HackConfig holds configuration for client hacks/cheats.
type HackConfig struct {
	Flying          bool
	NoClip          bool
	Speeding        bool
	SpawnControl    bool
	ThirdPersonView bool
	JumpHeight      int
}

// Level represents a level, which contains blocks and various metadata.
type Level struct {
	server *Server
	name   string

	width  uint
	height uint
	length uint
	Blocks []byte
	Dirty  bool

	UUID        [16]byte
	TimeCreated time.Time
	Metadata    map[string]interface{}

	MOTD       string
	Spawn      Location
	EnvConfig  EnvConfig
	HackConfig HackConfig
	BlockDefs  []*BlockDefinition
	Inventory  []byte
}

// NewLevel creates a new empty Level with the specified name and dimensions.
func NewLevel(name string, width, height, length uint) *Level {
	if len(name) == 0 {
		return nil
	}

	return &Level{
		name:        name,
		width:       width,
		height:      height,
		length:      length,
		Blocks:      make([]byte, width*height*length),
		Dirty:       true,
		UUID:        RandomUUID(),
		TimeCreated: time.Now(),
		Spawn: Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(length) / 2,
		},
		EnvConfig: EnvConfig{
			Weather:         WeatherSunny,
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

// Clone returns a duplicate of the level.
func (level Level) Clone(name string) *Level {
	if len(name) == 0 {
		return nil
	}

	newLevel := level
	newLevel.server = nil
	newLevel.name = name
	newLevel.Dirty = true
	newLevel.UUID = RandomUUID()
	newLevel.TimeCreated = time.Now()

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

// Size returns the number of blocks.
func (level *Level) Size() uint {
	return level.width * level.height * level.length
}

// Index converts the specified coordinates to an array index.
func (level *Level) Index(x, y, z uint) uint {
	return x + level.width*(z+level.length*y)
}

// Position converts the specified array index to block coordinates.
func (level *Level) Position(index uint) (x, y, z uint) {
	x = index % level.width
	y = (index / level.width) / level.length
	z = (index / level.width) % level.length
	return
}

// InBounds reports whether the specified coordinates are within the bounds of
// the level.
func (level *Level) InBounds(x, y, z uint) bool {
	return x < level.width && y < level.height && z < level.length
}

// GetBlock returns the block at the specified coordinates.
func (level *Level) GetBlock(x, y, z uint) byte {
	if x < level.width && y < level.height && z < level.length {
		return level.Blocks[level.Index(x, y, z)]
	}

	return BlockAir
}

// SetBlock sets the block at the specified coordinates.
// broadcast controls whether the block change is sent to the players.
func (level *Level) SetBlock(x, y, z uint, block byte, broadcast bool) {
	if x < level.width && y < level.height && z < level.length {
		level.Dirty = true
		level.Blocks[level.Index(x, y, z)] = block
		if broadcast {
			level.ForEachPlayer(func(player *Player) {
				player.sendBlockChange(x, y, z, block)
			})
		}
	}
}

// FillLayers fills the specified range of layers with block.
func (level *Level) FillLayers(yStart, yEnd uint, block byte) {
	start := yStart * level.width * level.length
	end := (yEnd + 1) * level.width * level.length
	for i := start; i < end; i++ {
		level.Blocks[i] = block
	}
}

// ForEachEntity calls fn for each entity in the level.
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

// ForEachPlayer calls fn for each player in the level.
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

// SendEnvConfig sends the EnvConfig of the level to all relevant players.
// mask controls which properties are sent.
func (level *Level) SendEnvConfig(mask uint32) {
	level.ForEachPlayer(func(player *Player) {
		player.sendEnvConfig(level, mask)
	})
}

// SendHackConfig sends the HackConfig of the level to all relevant players.
func (level *Level) SendHackConfig() {
	level.ForEachPlayer(func(player *Player) {
		player.sendHackConfig(level)
	})
}

// SendMOTD sends the MOTD of the level to all relevant players.
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

// BlockBuffer is a queue of block changes to apply to a level.
type BlockBuffer struct {
	level   *Level
	count   uint
	indices [256]int32
	blocks  [256]byte
}

// NewBlockBuffer returns a new BlockBuffer to queue changes to level.
func NewBlockBuffer(level *Level) *BlockBuffer {
	return &BlockBuffer{level: level}
}

// Set sets the block at the specified coordinates.
func (buffer *BlockBuffer) Set(x, y, z uint, block byte) {
	buffer.indices[buffer.count] = int32(buffer.level.Index(x, y, z))
	buffer.blocks[buffer.count] = block
	buffer.count++
	if buffer.count >= 256 {
		buffer.Flush()
	}
}

// Flush flushes any pending changes to the underlying level.
func (buffer *BlockBuffer) Flush() {
	for i := uint(0); i < buffer.count; i++ {
		index := buffer.indices[i]
		buffer.level.Blocks[index] = buffer.blocks[i]
	}

	buffer.level.Dirty = true
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
