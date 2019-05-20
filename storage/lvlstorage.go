// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package storage

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/structinf/Go-MCC/gomcc"
)

type lvlHeader struct {
	Version                          uint16
	Width, Height, Length            uint16
	SpawnX, SpawnY, SpawnZ           uint16
	SpawnYaw, SpawnPitch             byte
	PermissionVisit, PermissionBuild byte
}

type LvlStorage struct {
	dirPath string
}

func NewLvlStorage(dirPath string) *LvlStorage {
	os.Mkdir(dirPath, 0777)
	return &LvlStorage{dirPath}
}

func (storage *LvlStorage) getPath(name string) string {
	return storage.dirPath + name + ".lvl"
}

func (storage *LvlStorage) Load(name string) (level *gomcc.Level, err error) {
	file, err := os.Open(storage.getPath(name))
	if err != nil {
		return
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return
	}
	defer reader.Close()

	var header lvlHeader
	if err = binary.Read(reader, binary.BigEndian, &header); err != nil {
		return
	}

	if header.Version != 1874 {
		return nil, errors.New("lvlstorage: invalid format")
	}

	level = gomcc.NewLevel(name, uint(header.Width), uint(header.Height), uint(header.Length))
	if level == nil {
		return nil, errors.New("lvlstorage: level creation failed")
	}

	level.Spawn.X = float64(header.SpawnX) / 32
	level.Spawn.Y = float64(header.SpawnY) / 32
	level.Spawn.Z = float64(header.SpawnZ) / 32
	level.Spawn.Yaw = float64(header.SpawnYaw) * 360 / 256
	level.Spawn.Pitch = float64(header.SpawnPitch) * 360 / 256
	if _, err = io.ReadFull(reader, level.Blocks); err != nil {
		return nil, err
	}

	return
}

func (storage *LvlStorage) Save(level *gomcc.Level) (err error) {
	file, err := os.Create(storage.getPath(level.Name()))
	if err != nil {
		return
	}

	writer := gzip.NewWriter(file)
	defer file.Close()
	defer writer.Close()

	if err = binary.Write(writer, binary.BigEndian, lvlHeader{
		1874,
		uint16(level.Width()),
		uint16(level.Height()),
		uint16(level.Length()),
		uint16(level.Spawn.X * 32),
		uint16(level.Spawn.Y * 32),
		uint16(level.Spawn.Z * 32),
		byte(level.Spawn.Yaw * 256 / 360),
		byte(level.Spawn.Pitch * 256 / 360),
		0, 0,
	}); err != nil {
		return
	}

	_, err = writer.Write(level.Blocks)
	return
}
