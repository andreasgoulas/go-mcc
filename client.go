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
	{"ExtPlayerList", 2},
	{"LongerMessages", 1},
	{"ChangeModel", 1},
	{"EnvMapAppearance", 2},
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

type Client struct {
	Entity    *Entity
	Server    *Server
	Conn      net.Conn
	Connected uint32
	LoggedIn  uint32
	Name      string

	Operator bool

	HasCPE                  bool
	RemainingExtensions     uint
	Extensions              map[string]int
	Message                 string
	CustomBlockSupportLevel byte
	ClickDistance           float64

	PingTicker *time.Ticker
}

func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		Server:        server,
		Conn:          conn,
		Extensions:    make(map[string]int),
		ClickDistance: 5.0,
	}
}

func (client *Client) IsOperator() bool {
	return client.Operator
}

func (client *Client) Verify(key []byte) bool {
	if len(key) != md5.Size {
		return false
	}

	data := make([]byte, len(client.Server.Salt))
	copy(data, client.Server.Salt[:])
	data = append(data, []byte(client.Name)...)

	digest := md5.Sum(data)
	return bytes.Equal(digest[:], key)
}

func (client *Client) Handle() {
	buffer := make([]byte, 256)
	atomic.StoreUint32(&client.Connected, 1)
	for client.Connected == 1 {
		buffer = buffer[:1]
		_, err := io.ReadFull(client.Conn, buffer)
		if err != nil {
			client.Disconnect()
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
		_, err = io.ReadFull(client.Conn, buffer[1:])
		if err != nil {
			client.Disconnect()
			return
		}

		reader := bytes.NewReader(buffer)
		switch id {
		case PacketTypeIdentification:
			client.HandleIdentification(reader)
		case PacketTypeSetBlockClient:
			client.HandleSetBlock(reader)
		case PacketTypePlayerTeleport:
			client.HandlePlayerTeleport(reader)
		case PacketTypeMessage:
			client.HandleMessage(reader)
		case PacketTypeExtInfo:
			client.HandleExtInfo(reader)
		case PacketTypeExtEntry:
			client.HandleExtEntry(reader)
		case PacketTypeCustomBlockSupportLevel:
			client.HandleCustomBlockSupportLevel(reader)
		}
	}
}

func (client *Client) Login() {
	if client.LoggedIn == 1 {
		return
	}

	for {
		count := client.Server.PlayerCount
		if int(count) >= client.Server.Config.MaxPlayers {
			client.Kick("Server full!")
			return
		}

		if atomic.CompareAndSwapInt32(&client.Server.PlayerCount, count, count+1) {
			break
		}
	}

	if client.HasExtension("CustomBlocks") {
		client.SendPacket(&PacketCustomBlockSupportLevel{
			PacketTypeCustomBlockSupportLevel,
			1,
		})
	}

	userType := byte(0x00)
	if client.Operator {
		userType = 0x64
	}

	client.SendPacket(&PacketServerIdentification{
		PacketTypeIdentification,
		0x07,
		PadString(client.Server.Config.Name),
		PadString(client.Server.Config.MOTD),
		userType,
	})

	client.Entity = NewEntity(client.Name, client.Server)
	client.Entity.Client = client

	event := EventPlayerJoin{client.Entity, false, ""}
	client.Server.FireEvent(EventTypePlayerJoin, &event)
	if event.Cancel {
		client.Kick(event.CancelReason)
		return
	}

	atomic.StoreUint32(&client.LoggedIn, 1)
	client.Server.AddClient(client)
	client.Server.BroadcastMessage(ColorYellow + client.Entity.Name + " has joined the game!")
	if client.Server.AddEntity(client.Entity) == 0xff {
		client.Kick("Server full!")
		return
	}

	level := client.Server.MainLevel()
	if level != nil {
		client.Entity.TeleportLevel(level)
	}

	client.PingTicker = time.NewTicker(2 * time.Second)
	go func() {
		for range client.PingTicker.C {
			client.SendPacket(&PacketPing{PacketTypePing})
		}
	}()
}

func (client *Client) HandleIdentification(reader io.Reader) {
	if client.LoggedIn == 1 {
		return
	}

	packet := PacketClientIdentification{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.ProtocolVersion != 0x07 {
		client.Kick("Wrong version!")
		return
	}

	client.Name = TrimString(packet.Name)
	if !IsValidName(client.Name) {
		client.Kick("Invalid name!")
		return
	}

	key := TrimString(packet.VerificationKey)
	if client.Server.Config.Verify {
		if !client.Verify([]byte(key)) {
			client.Kick("Login failed!")
			return
		}
	}

	if client.Server.FindEntity(client.Name) != nil {
		client.Kick("Already logged in!")
		return
	}

	if packet.Type == 0x42 {
		client.HasCPE = true
		client.SendCPE()
	} else {
		client.Login()
	}
}

func (client *Client) RevertBlock(x, y, z uint) {
	client.SendBlockChange(x, y, z, client.Entity.Level.GetBlock(x, y, z))
}

func (client *Client) HandleSetBlock(reader io.Reader) {
	if client.LoggedIn == 0 {
		return
	}

	packet := PacketSetBlockClient{}
	binary.Read(reader, binary.BigEndian, &packet)
	x, y, z := uint(packet.X), uint(packet.Y), uint(packet.Z)
	block := BlockID(packet.BlockType)

	dx := uint(client.Entity.Location.X) - x
	dy := uint(client.Entity.Location.Y) - y
	dz := uint(client.Entity.Location.Z) - z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > client.ClickDistance {
		client.SendMessage("You can't build that far away.")
		client.RevertBlock(x, y, z)
		return
	}

	switch packet.Mode {
	case 0x00:
		event := &EventBlockBreak{
			client.Entity,
			client.Entity.Level,
			client.Entity.Level.GetBlock(x, y, z),
			x, y, z,
			false,
		}
		client.Server.FireEvent(EventTypeBlockBreak, &event)
		if event.Cancel {
			return
		}

		client.Entity.Level.SetBlock(x, y, z, BlockAir, client.Server)

	case 0x01:
		if block > BlockMaxCPE || (client.CustomBlockSupportLevel < 1 && block > BlockMax) {
			client.SendMessage("Invalid block!")
			client.RevertBlock(x, y, z)
			return
		}

		event := &EventBlockPlace{
			client.Entity,
			client.Entity.Level,
			block,
			x, y, z,
			false,
		}
		client.Server.FireEvent(EventTypeBlockPlace, &event)
		if event.Cancel {
			return
		}

		client.Entity.Level.SetBlock(x, y, z, block, client.Server)
	}
}

func (client *Client) HandlePlayerTeleport(reader io.Reader) {
	if client.LoggedIn == 0 {
		return
	}

	packet := PacketPlayerTeleport{}
	binary.Read(reader, binary.BigEndian, &packet)
	if packet.PlayerID != 0xff {
		return
	}

	client.Entity.Location = Location{
		float64(packet.X) / 32,
		float64(packet.Y) / 32,
		float64(packet.Z) / 32,
		float64(packet.Yaw) * 360 / 256,
		float64(packet.Pitch) * 360 / 256,
	}
}

func (client *Client) HandleMessage(reader io.Reader) {
	if client.LoggedIn == 0 {
		return
	}

	packet := PacketMessage{}
	binary.Read(reader, binary.BigEndian, &packet)

	client.Message += TrimString(packet.Message)
	if packet.PlayerID != 0x00 && client.HasExtension("LongerMessages") {
		return
	}

	message := client.Message
	client.Message = ""

	if !IsValidMessage(message) {
		client.SendMessage("Invalid message!")
		return
	}

	if message[0] == '/' {
		client.Server.ExecuteCommand(client, message[1:])
	} else {
		client.Server.BroadcastMessage(ColorDefault + "<" + client.Entity.Name + "> " + EscapeColors(message))
	}
}

func (client *Client) HandleExtInfo(reader io.Reader) {
	packet := PacketExtInfo{}
	binary.Read(reader, binary.BigEndian, &packet)

	client.RemainingExtensions = uint(packet.ExtensionCount)
	if client.RemainingExtensions == 0 {
		client.Login()
	}
}

func (client *Client) HandleExtEntry(reader io.Reader) {
	packet := PacketExtEntry{}
	binary.Read(reader, binary.BigEndian, &packet)

	for _, extension := range Extensions {
		if extension.Name == TrimString(packet.ExtName) {
			if extension.Version == int(packet.Version) {
				client.Extensions[extension.Name] = int(packet.Version)
				break
			}
		}
	}

	client.RemainingExtensions--
	if client.RemainingExtensions == 0 {
		client.Login()
	}
}

func (client *Client) HandleCustomBlockSupportLevel(reader io.Reader) {
	packet := PacketCustomBlockSupportLevel{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.SupportLevel <= 1 {
		client.CustomBlockSupportLevel = packet.SupportLevel
	}
}

func (client *Client) HasExtension(extension string) (f bool) {
	_, f = client.Extensions[extension]
	return
}

func (client *Client) Disconnect() {
	if client.Connected == 0 {
		return
	}
	atomic.StoreUint32(&client.Connected, 0)

	if client.PingTicker != nil {
		client.PingTicker.Stop()
	}

	client.Conn.Close()

	if client.LoggedIn == 1 {
		atomic.StoreUint32(&client.LoggedIn, 0)

		event := EventPlayerQuit{client.Entity}
		client.Server.FireEvent(EventTypePlayerQuit, &event)

		client.Entity.TeleportLevel(nil)
		client.Server.BroadcastMessage(ColorYellow + client.Entity.Name + " has left the game!")
		client.Server.RemoveClient(client)
		client.Server.RemoveEntity(client.Entity)
		atomic.AddInt32(&client.Server.PlayerCount, -1)
	}

	event := EventClientDisconnect{client}
	client.Server.FireEvent(EventTypeClientDisconnect, &event)
}

func (client *Client) SendPacket(packet interface{}) {
	if client.Connected == 0 {
		return
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, packet)
	_, err := buffer.WriteTo(client.Conn)
	if err == io.EOF {
		client.Disconnect()
	}
}

func (client *Client) Kick(reason string) {
	client.SendPacket(&PacketDisconnect{
		PacketTypeDisconnect,
		PadString(reason),
	})
	client.Disconnect()
}

func (client *Client) SendMessage(message string) {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		client.SendPacket(&PacketMessage{
			PacketTypeMessage,
			0x00,
			PadString(line),
		})
	}
}

func (client *Client) ConvertBlock(block BlockID) byte {
	if client.CustomBlockSupportLevel < 1 {
		return byte(FallbackBlock(block))
	}

	return byte(block)
}

func (client *Client) SendLevel(level *Level) {
	if client.LoggedIn == 0 {
		return
	}

	client.SendPacket(&PacketLevelInitialize{PacketTypeLevelInitialize})

	var GZIPBuffer bytes.Buffer
	GZIPWriter := gzip.NewWriter(&GZIPBuffer)
	binary.Write(GZIPWriter, binary.BigEndian, int32(level.Volume()))
	for _, block := range level.Blocks {
		GZIPWriter.Write([]byte{client.ConvertBlock(block)})
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
		client.SendPacket(packet)
	}

	if client.HasExtension("EnvMapAppearance") {
		client.SendPacket(&PacketEnvSetMapAppearance2{
			PacketTypeEnvSetMapAppearance2,
			PadString(level.Appearance.TexturePackURL),
			client.ConvertBlock(level.Appearance.SideBlock),
			client.ConvertBlock(level.Appearance.EdgeBlock),
			int16(level.Appearance.SideLevel),
			int16(level.Appearance.CloudLevel),
			int16(level.Appearance.MaxViewDistance),
		})
	}

	client.SendPacket(&PacketLevelFinalize{
		PacketTypeLevelFinalize,
		int16(level.Width), int16(level.Height), int16(level.Depth),
	})
}

func (client *Client) SendSpawn(entity *Entity) {
	if client.LoggedIn == 0 {
		return
	}

	id := entity.NameID
	if id == client.Entity.NameID {
		id = 0xff
	}

	location := entity.Location
	if client.HasExtension("ExtPlayerList") {
		client.SendPacket(&PacketExtAddEntity2{
			PacketTypeExtAddEntity2,
			id,
			PadString(entity.DisplayName),
			PadString(entity.SkinName),
			int16(location.X * 32),
			int16(location.Y * 32),
			int16(location.Z * 32),
			byte(location.Yaw * 256 / 360),
			byte(location.Pitch * 256 / 360),
		})
	} else {
		client.SendPacket(&PacketSpawnPlayer{
			PacketTypeSpawnPlayer,
			id,
			PadString(entity.Name),
			int16(location.X * 32),
			int16(location.Y * 32),
			int16(location.Z * 32),
			byte(location.Yaw * 256 / 360),
			byte(location.Pitch * 256 / 360),
		})
	}

	if entity.ModelName != ModelHumanoid {
		client.SendChangeModel(entity)
	}
}

func (client *Client) SendDespawn(entity *Entity) {
	if client.LoggedIn == 0 {
		return
	}

	id := entity.NameID
	if id == client.Entity.NameID {
		id = 0xff
	}

	client.SendPacket(&PacketDespawnPlayer{
		PacketTypeDespawnPlayer,
		id,
	})
}

func (client *Client) SendBlockChange(x, y, z uint, block BlockID) {
	if client.LoggedIn == 0 {
		return
	}

	client.SendPacket(&PacketSetBlock{
		PacketTypeSetBlock,
		int16(x), int16(y), int16(z),
		client.ConvertBlock(block),
	})
}

func (client *Client) SetOperator(value bool) {
	if client.LoggedIn == 1 && value != client.Operator {
		userType := byte(0x00)
		if value {
			userType = 0x64
		}

		client.SendPacket(&PacketUpdateUserType{
			PacketTypeUpdateUserType,
			userType,
		})
	}

	client.Operator = value
}

func (client *Client) SetClickDistance(value float64) {
	if client.LoggedIn == 1 && client.HasExtension("ClickDistance") {
		client.SendPacket(&PacketSetClickDistance{
			PacketTypeSetClickDistance,
			int16(value * 32),
		})
	}

	client.ClickDistance = value
}

func (client *Client) SendCPE() {
	client.SendPacket(&PacketExtInfo{
		PacketTypeExtInfo,
		PadString(ServerSoftware),
		int16(len(Extensions)),
	})

	for _, extension := range Extensions {
		client.SendPacket(&PacketExtEntry{
			PacketTypeExtEntry,
			PadString(extension.Name),
			int32(extension.Version),
		})
	}
}

func (client *Client) SendAddPlayerList(entity *Entity) {
	if client.LoggedIn == 0 || !client.HasExtension("ExtPlayerList") {
		return
	}

	id := entity.NameID
	if id == client.Entity.NameID {
		id = 0xff
	}

	client.SendPacket(&PacketExtAddPlayerName{
		PacketTypeExtAddPlayerName,
		int16(id),
		PadString(entity.Name),
		PadString(entity.ListName),
		PadString(entity.GroupName),
		entity.GroupRank,
	})
}

func (client *Client) SendRemovePlayerList(entity *Entity) {
	if client.LoggedIn == 0 || !client.HasExtension("ExtPlayerList") {
		return
	}

	id := entity.NameID
	if id == client.Entity.NameID {
		id = 0xff
	}

	client.SendPacket(&PacketExtRemovePlayerName{
		PacketTypeExtRemovePlayerName,
		int16(id),
	})
}

func (client *Client) SendChangeModel(entity *Entity) {
	if client.LoggedIn == 0 || !client.HasExtension("ChangeModel") {
		return
	}

	id := entity.NameID
	if id == client.Entity.NameID {
		id = 0xff
	}

	client.SendPacket(&PacketChangeModel{
		PacketTypeChangeModel,
		id,
		PadString(entity.ModelName),
	})
}
