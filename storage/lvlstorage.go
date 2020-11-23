package storage

import (
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/structinf/go-mcc/mcc"
)

type lvlHeader struct {
	Version                          int16
	Width, Height, Length            int16
	SpawnX, SpawnY, SpawnZ           int16
	SpawnYaw, SpawnPitch             byte
	PermissionVisit, PermissionBuild byte
}

// LvlStorage is an implementation of the mcc.LevelStorage interface that can
// handle MCSharp (.lvl) levels.
type LvlStorage struct {
	dirPath string
}

// NewLvlStorage creates a new LvlStorage that uses dirPath as the working
// directory.
func NewLvlStorage(dirPath string) *LvlStorage {
	os.Mkdir(dirPath, 0777)
	return &LvlStorage{dirPath}
}

func (storage *LvlStorage) getPath(name string) string {
	return storage.dirPath + name + ".lvl"
}

// Load implements mcc.LevelStorage.
func (storage *LvlStorage) Load(name string) (level *mcc.Level, err error) {
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

	level = mcc.NewLevel(name, int(header.Width), int(header.Height), int(header.Length))
	if level == nil {
		return nil, errors.New("lvlstorage: level creation failed")
	}

	level.Spawn.X = float64(header.SpawnX) + 0.5
	level.Spawn.Y = float64(header.SpawnY)
	level.Spawn.Z = float64(header.SpawnZ) + 0.5
	level.Spawn.Yaw = float64(header.SpawnYaw) * 360 / 256
	level.Spawn.Pitch = float64(header.SpawnPitch) * 360 / 256
	if _, err = io.ReadFull(reader, level.Blocks); err != nil {
		return nil, err
	}

	return
}

// Save implements mcc.LevelStorage.
func (storage *LvlStorage) Save(level *mcc.Level) (err error) {
	file, err := os.Create(storage.getPath(level.Name))
	if err != nil {
		return
	}

	writer := gzip.NewWriter(file)
	defer file.Close()
	defer writer.Close()

	if err = binary.Write(writer, binary.BigEndian, lvlHeader{
		1874,
		int16(level.Width),
		int16(level.Height),
		int16(level.Length),
		int16(level.Spawn.X),
		int16(level.Spawn.Y),
		int16(level.Spawn.Z),
		byte(level.Spawn.Yaw * 256 / 360),
		byte(level.Spawn.Pitch * 256 / 360),
		0, 0,
	}); err != nil {
		return
	}

	_, err = writer.Write(level.Blocks)
	return
}
