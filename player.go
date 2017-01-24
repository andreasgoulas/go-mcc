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

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

var Extensions = []struct {
	Name    string
	Version int
}{
	{"ClickDistance", 1},
	{"CustomBlocks", 1},
	{"LongerMessages", 1},
}

func IsValidName(name string) bool {
	if len(name) < 3 || len(name) > 16 {
		return false
	}

	for _, c := range name {
		if c > unicode.MaxASCII || (!unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_') {
			return false
		}
	}

	return true
}

func IsValidMessage(message string) bool {
	for _, c := range message {
		if c > unicode.MaxASCII || !unicode.IsPrint(c) || c == '&' {
			return false
		}
	}

	return true
}

func EscapeColors(message string) string {
	result := make([]byte, len(message))
	for i := range message {
		result[i] = message[i]
		if message[i] == '%' && i < len(message)-1 {
			color := message[i+1]
			if (color >= 'a' && color <= 'f') ||
				(color >= 'A' && color <= 'A') ||
				(color >= '0' && color <= '9') {
				result[i] = '&'
			}
		}
	}

	return string(result)
}

type Player struct {
	Name        string
	ID          byte
	Level       *Level
	Location    Location
	OldLocation Location

	Operator bool

	Server    *Server
	Conn      net.Conn
	Connected uint32
	LoggedIn  uint32

	HasCPE                  bool
	RemainingExtensions     uint
	Extensions              map[string]int
	Message                 string
	CustomBlockSupportLevel byte
	ClickDistance           float64

	PingTicker *time.Ticker
}

func NewPlayer(server *Server, conn net.Conn) *Player {
	return &Player{
		ID:            0xff,
		Server:        server,
		Conn:          conn,
		Extensions:    make(map[string]int),
		ClickDistance: 5.0,
	}
}

func (player *Player) GetName() string {
	return player.Name
}

func (player *Player) GetID() byte {
	return player.ID
}

func (player *Player) SetID(id byte) {
	player.ID = id
}

func (player *Player) GetLocation() Location {
	return player.Location
}

func (player *Player) GetLevel() *Level {
	return player.Level
}

func (player *Player) IsOperator() bool {
	return player.Operator
}

func (player *Player) Teleport(location Location) {
	player.Location = location
}

func (player *Player) TeleportLevel(level *Level) {
	if player.LoggedIn == 0 || level == player.Level {
		return
	}

	if player.Level != nil {
		level := player.Level
		player.Level = nil
		level.RemovePlayer(player)
	}

	player.Location = level.Spawn
	player.OldLocation = Location{}
	if !level.AddPlayer(player) {
		player.Kick("Level full!")
		return
	}

	player.Level = level
}

func (player *Player) Verify(key []byte) bool {
	if len(key) != md5.Size {
		return false
	}

	data := make([]byte, len(player.Server.Salt))
	copy(data, player.Server.Salt[:])
	data = append(data, []byte(player.Name)...)

	digest := md5.Sum(data)
	return bytes.Equal(digest[:], key)
}

func (player *Player) Handle() {
	buffer := make([]byte, 256)
	atomic.StoreUint32(&player.Connected, 1)
	for player.Connected == 1 {
		buffer = buffer[:1]
		_, err := io.ReadFull(player.Conn, buffer)
		if err != nil {
			player.Disconnect()
			return
		}

		id := buffer[0]
		var size uint
		switch id {
		case PacketTypeIdentification:
			size = 131
		case PacketTypeSetBlockClient:
			size = 9
		case PacketTypePlayerTeleport:
			size = 10
		case PacketTypeMessage:
			size = 66
		case PacketTypeExtInfo:
			size = 67
		case PacketTypeExtEntry:
			size = 69
		case PacketTypeCustomBlockSupportLevel:
			size = 2

		default:
			fmt.Printf("Invalid Packet: %d\n", id)
			continue
		}

		buffer = buffer[:size]
		_, err = io.ReadFull(player.Conn, buffer[1:])
		if err != nil {
			player.Disconnect()
			return
		}

		reader := bytes.NewReader(buffer)
		switch id {
		case PacketTypeIdentification:
			player.HandleIdentification(reader)

		case PacketTypeSetBlockClient:
			player.HandleSetBlock(reader)

		case PacketTypePlayerTeleport:
			player.HandlePlayerTeleport(reader)

		case PacketTypeMessage:
			player.HandleMessage(reader)

		case PacketTypeExtInfo:
			player.HandleExtInfo(reader)

		case PacketTypeExtEntry:
			player.HandleExtEntry(reader)

		case PacketTypeCustomBlockSupportLevel:
			player.HandleCustomBlockSupportLevel(reader)
		}
	}
}

func (player *Player) Login() {
	if player.LoggedIn == 1 {
		return
	}

	if player.HasExtension("CustomBlocks") {
		player.SendPacket(&PacketCustomBlockSupportLevel{
			PacketTypeCustomBlockSupportLevel,
			1,
		})
	}

	for {
		count := player.Server.PlayerCount
		if int(count) >= player.Server.Config.MaxPlayers {
			player.Kick("Server full!")
			return
		}

		if atomic.CompareAndSwapInt32(&player.Server.PlayerCount, count, count+1) {
			break
		}
	}

	player.SendLogin()
	atomic.StoreUint32(&player.LoggedIn, 1)
	player.Server.BroadcastMessage(ColorYellow + player.Name + " has joined the game!")

	level := player.Server.MainLevel()
	if level != nil {
		player.TeleportLevel(level)
	}

	player.PingTicker = time.NewTicker(2 * time.Second)
	go func() {
		for range player.PingTicker.C {
			player.SendPing()
		}
	}()
}

func (player *Player) HandleIdentification(reader io.Reader) {
	if player.LoggedIn == 1 {
		return
	}

	packet := PacketClientIdentification{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.ProtocolVersion != 0x07 {
		player.Kick("Wrong version!")
		return
	}

	player.Name = TrimString(packet.Name)
	if !IsValidName(player.Name) {
		player.Kick("Invalid name!")
		return
	}

	key := TrimString(packet.VerificationKey)
	if player.Server.Config.Verify {
		if !player.Verify([]byte(key)) {
			player.Kick("Login failed!")
			return
		}
	}

	if player.Server.FindPlayer(player.Name) != nil {
		player.Kick("Already logged in!")
		return
	}

	if packet.Type == 0x42 {
		player.HasCPE = true
		player.SendCPE()
	} else {
		player.Login()
	}
}

func (player *Player) RevertBlock(x, y, z uint) {
	player.SendBlockChange(x, y, z, player.Level.GetBlock(x, y, z))
}

func (player *Player) HandleSetBlock(reader io.Reader) {
	if player.LoggedIn == 0 {
		return
	}

	packet := PacketSetBlockClient{}
	binary.Read(reader, binary.BigEndian, &packet)
	x, y, z := uint(packet.X), uint(packet.Y), uint(packet.Z)
	block := BlockID(packet.BlockType)

	dx := uint(player.Location.X) - x
	dy := uint(player.Location.Y) - y
	dz := uint(player.Location.Z) - z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > player.ClickDistance {
		player.SendMessage("You can't build that far away.")
		player.RevertBlock(x, y, z)
		return
	}

	switch packet.Mode {
	case 0x00:
		player.Level.SetBlock(x, y, z, BlockAir, true)

	case 0x01:
		if block > BlockMaxCPE || (player.CustomBlockSupportLevel < 1 && block > BlockMax) {
			player.SendMessage("Invalid block!")
			player.RevertBlock(x, y, z)
			return
		}

		player.Level.SetBlock(x, y, z, block, true)
	}
}

func (player *Player) HandlePlayerTeleport(reader io.Reader) {
	if player.LoggedIn == 0 {
		return
	}

	packet := PacketPlayerTeleport{}
	binary.Read(reader, binary.BigEndian, &packet)
	if packet.PlayerID != 0xff {
		return
	}

	player.Location = Location{
		float64(packet.X) / 32,
		float64(packet.Y) / 32,
		float64(packet.Z) / 32,
		float64(packet.Yaw) * 360 / 256,
		float64(packet.Pitch) * 360 / 256,
	}
}

func (player *Player) HandleMessage(reader io.Reader) {
	if player.LoggedIn == 0 {
		return
	}

	packet := PacketMessage{}
	binary.Read(reader, binary.BigEndian, &packet)

	player.Message += TrimString(packet.Message)
	if packet.PlayerID != 0x00 && player.HasExtension("LongerMessages") {
		return
	}

	message := player.Message
	player.Message = ""

	if !IsValidMessage(message) {
		player.SendMessage("Invalid message!")
		return
	}

	if message[0] == '/' {
		player.Server.ExecuteCommand(player, message[1:])
	} else {
		player.Server.BroadcastMessage(ColorDefault + "<" + player.Name + "> " + EscapeColors(message))
	}
}

func (player *Player) HandleExtInfo(reader io.Reader) {
	packet := PacketExtInfo{}
	binary.Read(reader, binary.BigEndian, &packet)

	player.RemainingExtensions = uint(packet.ExtensionCount)
	if player.RemainingExtensions == 0 {
		player.Login()
	}
}

func (player *Player) HandleExtEntry(reader io.Reader) {
	packet := PacketExtEntry{}
	binary.Read(reader, binary.BigEndian, &packet)

	for _, extension := range Extensions {
		if extension.Name == TrimString(packet.ExtName) {
			if extension.Version == int(packet.Version) {
				player.Extensions[extension.Name] = int(packet.Version)
				break
			}
		}
	}

	player.RemainingExtensions--
	if player.RemainingExtensions == 0 {
		player.Login()
	}
}

func (player *Player) HandleCustomBlockSupportLevel(reader io.Reader) {
	packet := PacketCustomBlockSupportLevel{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.SupportLevel <= 1 {
		player.CustomBlockSupportLevel = packet.SupportLevel
	}
}

func (player *Player) HasExtension(extension string) (f bool) {
	_, f = player.Extensions[extension]
	return
}

func (player *Player) Disconnect() {
	if player.Connected == 0 {
		return
	}
	atomic.StoreUint32(&player.Connected, 0)

	if player.PingTicker != nil {
		player.PingTicker.Stop()
	}

	player.Conn.Close()

	if player.LoggedIn == 1 {
		atomic.StoreUint32(&player.LoggedIn, 0)
		if player.Level != nil {
			player.Level.RemovePlayer(player)
			atomic.AddInt32(&player.Server.PlayerCount, -1)
		}

		player.Server.BroadcastMessage(ColorYellow + player.Name + " has left the game!")
	}
}

func (player *Player) SendPacket(packet interface{}) {
	if player.Connected == 0 {
		return
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, packet)
	_, err := buffer.WriteTo(player.Conn)
	if err == io.EOF {
		player.Disconnect()
	}
}

func (player *Player) Kick(reason string) {
	player.SendPacket(&PacketDisconnect{
		PacketTypeDisconnect,
		PadString(reason),
	})
	player.Disconnect()
}

func (player *Player) SendPing() {
	player.SendPacket(&PacketPing{PacketTypePing})
}

func (player *Player) SendLogin() {
	userType := byte(0x00)
	if player.Operator {
		userType = 0x64
	}

	player.SendPacket(&PacketServerIdentification{
		PacketTypeIdentification,
		0x07,
		PadString(player.Server.Config.Name),
		PadString(player.Server.Config.MOTD),
		userType,
	})
}

func (player *Player) SendMessage(message string) {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		player.SendPacket(&PacketMessage{
			PacketTypeMessage,
			0x00,
			PadString(line),
		})
	}
}

func (player *Player) SendLevel(level *Level) {
	player.SendPacket(&PacketLevelInitialize{PacketTypeLevelInitialize})

	var GZIPBuffer bytes.Buffer
	GZIPWriter := gzip.NewWriter(&GZIPBuffer)
	binary.Write(GZIPWriter, binary.BigEndian, int32(level.Volume()))
	for _, block := range level.Blocks {
		client := byte(block)
		if player.CustomBlockSupportLevel < 1 {
			client = byte(Convert(block))
		}

		GZIPWriter.Write([]byte{client})
	}
	GZIPWriter.Close()

	GZIPData := GZIPBuffer.Bytes()
	packets := int(math.Ceil(float64(len(GZIPData)) / 1024))
	for i := 0; i < packets; i++ {
		offset := 1024 * i
		size := len(GZIPData) - offset
		if size > 1024 {
			size = 1024
		}

		packet := &PacketLevelDataChunk{
			PacketTypeLevelDataChunk,
			int16(size),
			[1024]byte{},
			byte(i * 100 / packets),
		}

		copy(packet.ChunkData[:], GZIPData[offset:offset+size])
		player.SendPacket(packet)
	}

	player.SendPacket(&PacketLevelFinalize{
		PacketTypeLevelFinalize,
		int16(level.Width), int16(level.Height), int16(level.Depth),
	})
}

func (player *Player) SendSpawn(entity Entity) {
	id := entity.GetID()
	if id == player.ID {
		id = 0xff
	}

	location := entity.GetLocation()
	player.SendPacket(&PacketSpawnPlayer{
		PacketTypeSpawnPlayer,
		id,
		PadString(entity.GetName()),
		int16(location.X * 32),
		int16(location.Y * 32),
		int16(location.Z * 32),
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (player *Player) SendDespawn(entity Entity) {
	id := entity.GetID()
	if id == player.ID {
		id = 0xff
	}

	player.SendPacket(&PacketDespawnPlayer{
		PacketTypeDespawnPlayer,
		id,
	})
}

func (player *Player) SendBlockChange(x, y, z uint, block BlockID) {
	client := byte(block)
	if player.CustomBlockSupportLevel < 1 {
		client = byte(Convert(block))
	}

	player.SendPacket(&PacketSetBlock{
		PacketTypeSetBlock,
		int16(x), int16(y), int16(z),
		client,
	})
}

func (player *Player) Update(dt time.Duration) {
	if player.Level == nil {
		return
	}

	positionDirty := false
	if player.Location.X != player.OldLocation.X ||
		player.Location.Y != player.OldLocation.Y ||
		player.Location.Z != player.OldLocation.Z {
		positionDirty = true
	}

	rotationDirty := false
	if player.Location.Yaw != player.OldLocation.Yaw ||
		player.Location.Pitch != player.OldLocation.Pitch {
		rotationDirty = true
	}

	teleport := false
	if math.Abs(player.Location.X-player.OldLocation.X) > 1.0 ||
		math.Abs(player.Location.Y-player.OldLocation.Y) > 1.0 ||
		math.Abs(player.Location.Z-player.OldLocation.Z) > 1.0 {
		teleport = true
	}

	var packet interface{}
	if teleport {
		packet = &PacketPlayerTeleport{
			PacketTypePlayerTeleport,
			player.ID,
			int16(player.Location.X * 32),
			int16(player.Location.Y * 32),
			int16(player.Location.Z * 32),
			byte(player.Location.Yaw * 256 / 360),
			byte(player.Location.Pitch * 256 / 360),
		}
	} else if positionDirty && rotationDirty {
		packet = &PacketPositionOrientationUpdate{
			PacketTypePositionOrientationUpdate,
			player.ID,
			byte((player.Location.X - player.OldLocation.X) * 32),
			byte((player.Location.Y - player.OldLocation.Y) * 32),
			byte((player.Location.Z - player.OldLocation.Z) * 32),
			byte(player.Location.Yaw * 256 / 360),
			byte(player.Location.Pitch * 256 / 360),
		}
	} else if positionDirty {
		packet = &PacketPositionUpdate{
			PacketTypePositionUpdate,
			player.ID,
			byte((player.Location.X - player.OldLocation.X) * 32),
			byte((player.Location.Y - player.OldLocation.Y) * 32),
			byte((player.Location.Z - player.OldLocation.Z) * 32),
		}
	} else if rotationDirty {
		packet = &PacketOrientationUpdate{
			PacketTypeOrientationUpdate,
			player.ID,
			byte(player.Location.Yaw * 256 / 360),
			byte(player.Location.Pitch * 256 / 360),
		}
	} else {
		return
	}

	level := player.Level
	level.PlayersLock.RLock()
	for _, other := range level.Players {
		if other != player {
			other.SendPacket(packet)
		}
	}
	level.PlayersLock.RUnlock()

	player.OldLocation = player.Location
}

func (player *Player) SetOperator(value bool) {
	if player.LoggedIn == 1 && value != player.Operator {
		userType := byte(0x00)
		if value {
			userType = 0x64
		}

		player.SendPacket(&PacketUpdateUserType{
			PacketTypeUpdateUserType,
			userType,
		})
	}

	player.Operator = value
}

func (player *Player) SetClickDistance(value float64) {
	if player.LoggedIn == 1 && player.HasExtension("ClickDistance") {
		player.SendPacket(&PacketSetClickDistance{
			PacketTypeSetClickDistance,
			int16(value * 32),
		})
	}

	player.ClickDistance = value
}

func (player *Player) SendCPE() {
	player.SendPacket(&PacketExtInfo{
		PacketTypeExtInfo,
		PadString(ServerSoftware),
		int16(len(Extensions)),
	})

	for _, extension := range Extensions {
		player.SendPacket(&PacketExtEntry{
			PacketTypeExtEntry,
			PadString(extension.Name),
			int32(extension.Version),
		})
	}
}
