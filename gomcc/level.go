// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

import (
	"image/color"
	"sync"
	"time"
)

// LevelStorage is the interface that must be implemented by storage backends
// that can import and export levels.
type LevelStorage interface {
	Load(name string) (*Level, error)
	Save(level *Level) error
}

// Simulator is the interface that must be implemented by block-based physics
// simulators.
type Simulator interface {
	Update(block, old byte, index int)
	Tick()
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
	EdgeHeight      int
	CloudHeight     int
	MaxViewDistance int
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
	ReachDistance   float64
	Flying          bool
	NoClip          bool
	Speeding        bool
	SpawnControl    bool
	ThirdPersonView bool
	JumpHeight      int

	CanPlace [BlockCount]bool
	CanBreak [BlockCount]bool
}

// Level represents a level, which contains blocks and various metadata.
type Level struct {
	server *Server

	Width  int
	Height int
	Length int
	Blocks []byte
	Dirty  bool

	Name        string
	UUID        [16]byte
	TimeCreated time.Time
	MOTD        string
	Spawn       Location
	EnvConfig   EnvConfig
	HackConfig  HackConfig
	BlockDefs   []*BlockDefinition
	Inventory   []byte

	Metadata, MetadataCPE map[string]interface{}

	simulators     []Simulator
	simulatorsLock sync.RWMutex
}

// NewLevel creates a new empty Level with the specified name and dimensions.
func NewLevel(name string, width, height, length int) *Level {
	if len(name) == 0 {
		return nil
	}

	level := &Level{
		Width:       width,
		Height:      height,
		Length:      length,
		Blocks:      make([]byte, width*height*length),
		Dirty:       true,
		Name:        name,
		UUID:        RandomUUID(),
		TimeCreated: time.Now(),
		Spawn: Location{
			X: float64(width) / 2,
			Y: float64(height) * 3 / 4,
			Z: float64(length) / 2,
		},
	}

	level.EnvConfig = level.DefaultEnvConfig()
	level.HackConfig = level.DefaultHackConfig()
	return level
}

// Clone returns a duplicate of the level.
func (level Level) Clone(name string) *Level {
	if len(name) == 0 {
		return nil
	}

	newLevel := level
	newLevel.server = nil
	newLevel.Name = name
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

// DefaultEnvConfig returns the default EnvConfig for this level.
func (level *Level) DefaultEnvConfig() EnvConfig {
	return EnvConfig{
		Weather:         WeatherSunny,
		SideBlock:       BlockBedrock,
		EdgeBlock:       BlockActiveWater,
		EdgeHeight:      level.Height / 2,
		CloudHeight:     level.Height + 2,
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
	}
}

// DefaultHackConfig returns the default HackConfig for this level.
func (level *Level) DefaultHackConfig() HackConfig {
	config := HackConfig{
		ReachDistance:   5,
		Flying:          false,
		NoClip:          false,
		Speeding:        false,
		SpawnControl:    true,
		ThirdPersonView: true,
		JumpHeight:      -1,
	}

	for i := 0; i < BlockCount; i++ {
		config.CanPlace[i] = true
		config.CanBreak[i] = true
	}

	banned := []byte{BlockBedrock, BlockActiveWater, BlockWater, BlockActiveLava, BlockLava}
	for _, block := range banned {
		config.CanPlace[block] = false
		config.CanBreak[block] = false
	}

	return config
}

// Size returns the number of blocks.
func (level *Level) Size() int {
	return level.Width * level.Height * level.Length
}

// Index converts the specified coordinates to an array index.
func (level *Level) Index(x, y, z int) int {
	return x + level.Width*(z+level.Length*y)
}

// Position converts the specified array index to block coordinates.
func (level *Level) Position(index int) (x, y, z int) {
	x = index % level.Width
	y = (index / level.Width) / level.Length
	z = (index / level.Width) % level.Length
	return
}

// InBounds reports whether the specified coordinates are within the bounds of
// the level.
func (level *Level) InBounds(x, y, z int) bool {
	return x < level.Width && y < level.Height && z < level.Length
}

// GetBlock returns the block at the specified coordinates.
func (level *Level) GetBlock(x, y, z int) byte {
	if x < level.Width && y < level.Height && z < level.Length {
		return level.Blocks[level.Index(x, y, z)]
	}

	return BlockAir
}

// SetBlockFast sets the block at the specified coordinates without notifying
// the physics simulators.
func (level *Level) SetBlockFast(x, y, z int, block byte) {
	if level.InBounds(x, y, z) {
		level.Dirty = true
		level.Blocks[level.Index(x, y, z)] = block
		level.ForEachPlayer(func(player *Player) {
			player.sendBlockChange(x, y, z, block)
		})
	}
}

// SetBlock sets the block at the specified coordinates.
func (level *Level) SetBlock(x, y, z int, block byte) {
	if level.InBounds(x, y, z) {
		index := level.Index(x, y, z)
		old := level.Blocks[index]

		level.Dirty = true
		level.Blocks[index] = block
		level.ForEachPlayer(func(player *Player) {
			player.sendBlockChange(x, y, z, block)
		})

		level.simulatorsLock.RLock()
		for _, simulator := range level.simulators {
			simulator.Update(block, old, index)
		}
		level.simulatorsLock.RUnlock()

		if x < level.Width-1 {
			level.UpdateBlock(x+1, y, z)
		}
		if x > 0 {
			level.UpdateBlock(x-1, y, z)
		}
		if y < level.Height-1 {
			level.UpdateBlock(x, y+1, z)
		}
		if y > 0 {
			level.UpdateBlock(x, y-1, z)
		}
		if z < level.Length-1 {
			level.UpdateBlock(x, y, z+1)
		}
		if z > 0 {
			level.UpdateBlock(x, y, z-1)
		}
	}
}

// FillLayers fills the specified range of layers with block.
func (level *Level) FillLayers(yStart, yEnd int, block byte) {
	start := yStart * level.Width * level.Length
	end := (yEnd + 1) * level.Width * level.Length
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

// RegisterSimulator registers a physics simulator.
func (level *Level) RegisterSimulator(simulator Simulator) {
	level.simulatorsLock.Lock()
	level.simulators = append(level.simulators, simulator)
	level.simulatorsLock.Unlock()

	for index, block := range level.Blocks {
		simulator.Update(block, block, index)
	}
}

// UnregisterSimulator unregisters a physics simulator.
func (level *Level) UnregisterSimulator(simulator Simulator) {
	level.simulatorsLock.Lock()
	defer level.simulatorsLock.Unlock()

	index := -1
	for i, s := range level.simulators {
		if s == simulator {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	level.simulators[index] = level.simulators[len(level.simulators)-1]
	level.simulators[len(level.simulators)-1] = nil
	level.simulators = level.simulators[:len(level.simulators)-1]
}

// UpdateBlock updates the block at the specified coordinates.
func (level *Level) UpdateBlock(x, y, z int) {
	index := level.Index(x, y, z)
	block := level.Blocks[index]
	level.simulatorsLock.RLock()
	for _, simulator := range level.simulators {
		simulator.Update(block, block, index)
	}
	level.simulatorsLock.RUnlock()
}

func (level *Level) update() {
	level.simulatorsLock.RLock()
	for _, simulator := range level.simulators {
		simulator.Tick()
	}
	level.simulatorsLock.RUnlock()
}

// BlockBuffer is a queue of block changes to apply to a level.
type BlockBuffer struct {
	level   *Level
	count   int
	indices [256]int32
	blocks  [256]byte
}

// NewBlockBuffer returns a new BlockBuffer to queue changes to level.
func NewBlockBuffer(level *Level) *BlockBuffer {
	return &BlockBuffer{level: level}
}

// Set sets the block at the specified coordinates.
func (buffer *BlockBuffer) Set(x, y, z int, block byte) {
	buffer.indices[buffer.count] = int32(buffer.level.Index(x, y, z))
	buffer.blocks[buffer.count] = block
	buffer.count++
	if buffer.count >= 256 {
		buffer.Flush()
	}
}

// Flush flushes any pending changes to the underlying level.
func (buffer *BlockBuffer) Flush() {
	for i := 0; i < buffer.count; i++ {
		index := buffer.indices[i]
		buffer.level.Blocks[index] = buffer.blocks[i]
	}

	buffer.level.Dirty = true
	buffer.level.ForEachPlayer(func(player *Player) {
		var blocks [256]byte
		for i := 0; i < buffer.count; i++ {
			blocks[i] = player.convertBlock(buffer.blocks[i], buffer.level)
		}

		var packet Packet
		if player.cpe[CpeBulkBlockUpdate] {
			packet.bulkBlockUpdate(buffer.indices[:], blocks[:buffer.count])
		} else {
			for i := 0; i < buffer.count; i++ {
				x, y, z := buffer.level.Position(int(buffer.indices[i]))
				packet.setBlock(x, y, z, blocks[i])
			}
		}

		player.sendPacket(packet)
	})

	buffer.count = 0
}
