// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package storage

import (
	"compress/gzip"
	"errors"
	"os"
	"time"

	"github.com/structinf/Go-MCC/gomcc"
)

type cwSpawn struct {
	X, Y, Z int16
	H, P    byte
}

type cwLevel struct {
	FormatVersion byte
	Name          string
	UUID          []byte
	X, Y, Z       int16
	TimeCreated   int64
	Spawn         cwSpawn
	BlockArray    []byte
}

type CwStorage struct {
	dirPath string
}

func NewCwStorage(dirPath string) *CwStorage {
	os.Mkdir(dirPath, 0777)
	return &CwStorage{dirPath}
}

func (storage *CwStorage) getPath(name string) string {
	return storage.dirPath + name + ".cw"
}

func (storage *CwStorage) Load(name string) (level *gomcc.Level, err error) {
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

	level = gomcc.NewLevel(cw.Name, uint(cw.X), uint(cw.Y), uint(cw.Z))
	level.Spawn.X = float64(cw.Spawn.X) / 32
	level.Spawn.Y = float64(cw.Spawn.Y) / 32
	level.Spawn.Z = float64(cw.Spawn.Z) / 32
	level.Spawn.Yaw = float64(cw.Spawn.H) * 360 / 256
	level.Spawn.Pitch = float64(cw.Spawn.P) * 360 / 256
	copy(level.Blocks[:], cw.BlockArray)
	copy(level.UUID[:], cw.UUID)

	if cw.TimeCreated > 0 {
		level.TimeCreated = time.Unix(cw.TimeCreated, 0)
	} else if stat, err := os.Stat(path); err != nil {
		level.TimeCreated = stat.ModTime()
	}

	return
}

func (storage *CwStorage) Save(level *gomcc.Level) (err error) {
	file, err := os.Create(storage.getPath(level.Name()))
	if err != nil {
		return
	}

	writer := gzip.NewWriter(file)
	defer file.Close()
	defer writer.Close()

	return NbtMarshal(writer, "ClassicWorld", cwLevel{
		1,
		level.Name(),
		level.UUID[:],
		int16(level.Width()),
		int16(level.Height()),
		int16(level.Length()),
		level.TimeCreated.Unix(),
		cwSpawn{
			int16(level.Spawn.X * 32),
			int16(level.Spawn.Y * 32),
			int16(level.Spawn.Z * 32),
			byte(level.Spawn.Yaw * 256 / 360),
			byte(level.Spawn.Pitch * 256 / 360),
		},
		level.Blocks,
	})
}
