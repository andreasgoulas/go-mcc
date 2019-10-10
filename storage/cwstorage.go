// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package storage

import (
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/structinf/go-mcc/mcc"
)

type cwSpawn struct {
	X, Y, Z int16
	H, P    byte
}

type cwClickDistance struct {
	ExtensionVersion int16
	Distance         int16
}

type cwColor struct {
	R, G, B int16
}

func encodeColor(c mcc.NullRGB) cwColor {
	if c.Valid {
		return cwColor{int16(c.R), int16(c.G), int16(c.B)}
	} else {
		return cwColor{-1, -1, -1}
	}
}

func (c cwColor) decode() mcc.NullRGB {
	if c.R < 0 || c.G < 0 || c.B < 0 {
		return mcc.NullRGB{}
	} else {
		return mcc.NullRGB{true, byte(c.R), byte(c.G), byte(c.B)}
	}
}

type cwEnvColors struct {
	ExtensionVersion                   int32
	Sky, Cloud, Fog, Ambient, Sunlight cwColor
}

type cwEnvMapAppearance struct {
	ExtensionVersion int32
	TextureURL       string
	SideBlock        byte
	EdgeBlock        byte
	SideLevel        int16
}

type cwEnvWeatherType struct {
	ExtensionVersion int32
	WeatherType      byte
}

type cwBlockDefinition struct {
	ID             byte
	Name           string
	Speed          float32
	CollideType    byte
	Textures       []byte
	TransmitsLight byte
	FullBright     byte
	WalkSound      byte
	Shape          byte
	BlockDraw      byte
	Fog            []byte
	Coords         []byte
}

var cwFaceIndices = []int{
	mcc.FacePosY, mcc.FaceNegY,
	mcc.FaceNegX, mcc.FacePosX,
	mcc.FaceNegZ, mcc.FacePosZ,
}

type CwBlockDefinitionMap map[string]cwBlockDefinition

type cwBlockDefinitions struct {
	ExtensionVersion int32
	CwBlockDefinitionMap
}

type CwMetadataMap map[string]interface{}

type cwCPE struct {
	CwMetadataMap
	ClickDistance    cwClickDistance
	EnvColors        cwEnvColors
	EnvMapAppearance cwEnvMapAppearance
	EnvWeatherType   cwEnvWeatherType
	BlockDefinitions cwBlockDefinitions
}

type cwMetadata struct {
	CwMetadataMap
	CPE cwCPE
}

type cwLevel struct {
	FormatVersion byte
	Name          string
	UUID          []byte
	X, Y, Z       int16
	TimeCreated   int64
	Spawn         cwSpawn
	BlockArray    []byte
	Metadata      cwMetadata
}

// CwStorage is an implementation of the mcc.levelStorage interface that can
// handle ClassicWorld (.cw) levels.
type CwStorage struct {
	dirPath string

	// FixSpawnPosition controls whether to attempt to parse the spawn
	// position as block coordinates. This format is incorrectly used by
	// some client software.
	FixSpawnPosition bool
}

// NewCwStorage creates a new CwStorage that uses dirPath as the working
// directory.
func NewCwStorage(dirPath string) *CwStorage {
	os.Mkdir(dirPath, 0777)
	return &CwStorage{dirPath, true}
}

func (storage *CwStorage) getPath(name string) string {
	return storage.dirPath + name + ".cw"
}

// Load implements mcc.LevelStorage.
func (storage *CwStorage) Load(name string) (level *mcc.Level, err error) {
	path := storage.getPath(name)
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return
	}
	defer reader.Close()

	var nbt struct{ ClassicWorld cwLevel }
	if err = NbtUnmarshal(reader, &nbt); err != nil {
		return
	}

	cw := &nbt.ClassicWorld
	if cw.FormatVersion != 1 {
		return nil, errors.New("cwstorage: invalid format")
	}

	level = mcc.NewLevel(name, int(cw.X), int(cw.Y), int(cw.Z))
	if level == nil {
		return nil, errors.New("cwstorage: level creation failed")
	}

	if storage.FixSpawnPosition &&
		cw.Spawn.X < cw.X && cw.Spawn.Y < cw.Y && cw.Spawn.Z < cw.Z {
		level.Spawn.X = float64(cw.Spawn.X) + 0.5
		level.Spawn.Y = float64(cw.Spawn.Y)
		level.Spawn.Z = float64(cw.Spawn.Z) + 0.5
	} else {
		level.Spawn.X = float64(cw.Spawn.X) / 32
		level.Spawn.Y = float64(cw.Spawn.Y) / 32
		level.Spawn.Z = float64(cw.Spawn.Z) / 32
	}

	level.Spawn.Yaw = float64(cw.Spawn.H) * 360 / 256
	level.Spawn.Pitch = float64(cw.Spawn.P) * 360 / 256
	copy(level.UUID[:], cw.UUID)

	if len(cw.BlockArray) == level.Size() {
		level.Blocks = cw.BlockArray
	}

	if cw.TimeCreated > 0 {
		level.TimeCreated = time.Unix(cw.TimeCreated, 0)
	} else if stat, err := os.Stat(path); err != nil {
		level.TimeCreated = stat.ModTime()
	}

	cpe := cw.Metadata.CPE
	if cpe.ClickDistance.ExtensionVersion == 1 {
		level.HackConfig.ReachDistance = float64(cpe.ClickDistance.Distance) / 32
	}

	if cpe.EnvColors.ExtensionVersion == 1 {
		level.EnvConfig.SkyColor = cpe.EnvColors.Sky.decode()
		level.EnvConfig.CloudColor = cpe.EnvColors.Cloud.decode()
		level.EnvConfig.FogColor = cpe.EnvColors.Fog.decode()
		level.EnvConfig.AmbientColor = cpe.EnvColors.Ambient.decode()
		level.EnvConfig.DiffuseColor = cpe.EnvColors.Sunlight.decode()
	}

	if cpe.EnvMapAppearance.ExtensionVersion == 1 {
		level.EnvConfig.TexturePack = cpe.EnvMapAppearance.TextureURL
		level.EnvConfig.SideBlock = cpe.EnvMapAppearance.SideBlock
		level.EnvConfig.EdgeBlock = cpe.EnvMapAppearance.EdgeBlock
		level.EnvConfig.EdgeHeight = int(cpe.EnvMapAppearance.SideLevel)
	}

	if cpe.EnvWeatherType.ExtensionVersion == 1 {
		level.EnvConfig.Weather = cpe.EnvWeatherType.WeatherType
	}

	if cpe.BlockDefinitions.ExtensionVersion == 1 {
		count := 0
		for _, v := range cpe.BlockDefinitions.CwBlockDefinitionMap {
			if int(v.ID) >= count {
				count = int(v.ID) + 1
			}
		}

		if count > 0 {
			level.BlockDefs = make([]*mcc.BlockDefinition, count)
		}

		for _, v := range cpe.BlockDefinitions.CwBlockDefinitionMap {
			def := &mcc.BlockDefinition{
				Name:        v.Name,
				Speed:       float64(v.Speed),
				CollideMode: v.CollideType,
				WalkSound:   v.WalkSound,
				BlockLight:  v.TransmitsLight == 0,
				FullBright:  v.FullBright == 1,
				DrawMode:    v.BlockDraw,
				Shape:       v.Shape,
			}

			if len(v.Textures) >= 6 {
				extTex := len(v.Textures) >= 12
				for i, face := range cwFaceIndices {
					def.Textures[face] = int(v.Textures[i])
					if extTex {
						def.Textures[face] |= int(v.Textures[i+6]) << 8
					}
				}
			}

			if len(v.Fog) == 4 {
				def.FogDensity = v.Fog[0]
				def.Fog = mcc.RGB{v.Fog[1], v.Fog[2], v.Fog[3]}
			}

			if len(v.Coords) == 6 {
				def.AABB = mcc.AABB{
					mcc.Vector3{int(v.Coords[0]), int(v.Coords[1]), int(v.Coords[2])},
					mcc.Vector3{int(v.Coords[3]), int(v.Coords[4]), int(v.Coords[5])},
				}
			}

			level.BlockDefs[v.ID] = def
		}
	}

	level.Metadata = cw.Metadata.CwMetadataMap
	level.MetadataCPE = cw.Metadata.CPE.CwMetadataMap
	return
}

// Save implements mcc.LevelStorage.
func (storage *CwStorage) Save(level *mcc.Level) (err error) {
	file, err := os.Create(storage.getPath(level.Name))
	if err != nil {
		return
	}

	writer := gzip.NewWriter(file)
	defer file.Close()
	defer writer.Close()

	var defs CwBlockDefinitionMap
	if level.BlockDefs != nil {
		defs = make(CwBlockDefinitionMap)
	}

	for i, v := range level.BlockDefs {
		if v != nil {
			def := cwBlockDefinition{
				ID:             byte(i),
				Name:           v.Name,
				Speed:          float32(v.Speed),
				CollideType:    v.CollideMode,
				Textures:       make([]byte, 12),
				TransmitsLight: 1,
				FullBright:     0,
				WalkSound:      v.WalkSound,
				Shape:          v.Shape,
				BlockDraw:      v.DrawMode,
				Fog:            []byte{v.FogDensity, v.Fog.R, v.Fog.G, v.Fog.B},
				Coords: []byte{
					byte(v.AABB.Min.X), byte(v.AABB.Min.Y), byte(v.AABB.Min.Z),
					byte(v.AABB.Max.X), byte(v.AABB.Max.Y), byte(v.AABB.Max.Z),
				},
			}

			for i, face := range cwFaceIndices {
				def.Textures[i] = byte(v.Textures[face])
				def.Textures[i+6] = byte(v.Textures[face] >> 8)
			}

			if v.BlockLight {
				def.TransmitsLight = 0
			}

			if v.FullBright {
				def.FullBright = 1
			}

			key := fmt.Sprintf("Block%d", i)
			defs[key] = def
		}
	}

	cpe := cwCPE{
		level.MetadataCPE,
		cwClickDistance{1, int16(level.HackConfig.ReachDistance * 32)},
		cwEnvColors{
			1,
			encodeColor(level.EnvConfig.SkyColor),
			encodeColor(level.EnvConfig.CloudColor),
			encodeColor(level.EnvConfig.FogColor),
			encodeColor(level.EnvConfig.AmbientColor),
			encodeColor(level.EnvConfig.DiffuseColor),
		},
		cwEnvMapAppearance{
			1,
			level.EnvConfig.TexturePack,
			level.EnvConfig.SideBlock,
			level.EnvConfig.EdgeBlock,
			int16(level.EnvConfig.EdgeHeight),
		},
		cwEnvWeatherType{1, level.EnvConfig.Weather},
		cwBlockDefinitions{1, defs},
	}

	return NbtMarshal(writer, "ClassicWorld", cwLevel{
		1,
		level.Name,
		level.UUID[:],
		int16(level.Width),
		int16(level.Height),
		int16(level.Length),
		level.TimeCreated.Unix(),
		cwSpawn{
			int16(level.Spawn.X * 32),
			int16(level.Spawn.Y * 32),
			int16(level.Spawn.Z * 32),
			byte(level.Spawn.Yaw * 256 / 360),
			byte(level.Spawn.Pitch * 256 / 360),
		},
		level.Blocks,
		cwMetadata{
			level.Metadata,
			cpe,
		},
	})
}
