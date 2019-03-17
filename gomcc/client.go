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
)

var Extensions = []struct {
	Name    string
	Version int
}{
	{"ClickDistance", 1},
	{"CustomBlocks", 1},
	{"HeldBlock", 1},
	{"ExtPlayerList", 2},
	{"LongerMessages", 1},
	{"ChangeModel", 1},
	{"EnvWeatherType", 1},
	{"PlayerClick", 1},
	{"EnvMapAspect", 1},
}

type Client struct {
	NickName string

	entity *Entity
	server *Server

	conn      net.Conn
	connected uint32
	loggedIn  uint32
	name      string

	operator    bool
	permissions [][]string

	remainingExtensions     uint
	extensions              map[string]int
	message                 string
	customBlockSupportLevel byte
	clickDistance           float64
	heldBlock               BlockID

	pingTicker *time.Ticker
}

func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		server:        server,
		conn:          conn,
		extensions:    make(map[string]int),
		clickDistance: 5.0,
	}
}

func (client *Client) Server() *Server {
	return client.server
}

func (client *Client) Entity() *Entity {
	return client.entity
}

func (client *Client) Name() string {
	return client.NickName
}

func (client *Client) checkPermission(permission []string, template []string) bool {
	lenP := len(permission)
	lenT := len(template)
	for i := 0; i < min(lenP, lenT); i++ {
		if template[i] == "*" {
			return true
		} else if permission[i] != template[i] {
			return false
		}
	}

	return lenP == lenT
}

func (client *Client) HasPermission(permission string) bool {
	if len(permission) == 0 {
		return true
	}

	split := strings.Split(permission, ".")
	for _, template := range client.permissions {
		if client.checkPermission(split, template) {
			return true
		}
	}

	return false
}

func (client *Client) HasExtension(extension string) (f bool) {
	_, f = client.extensions[extension]
	return
}

func (client *Client) Disconnect() {
	if client.connected == 0 {
		return
	}
	atomic.StoreUint32(&client.connected, 0)

	if client.pingTicker != nil {
		client.pingTicker.Stop()
	}

	client.conn.Close()

	if client.loggedIn == 1 {
		atomic.StoreUint32(&client.loggedIn, 0)

		event := EventPlayerQuit{client.entity}
		client.server.FireEvent(EventTypePlayerQuit, &event)

		client.entity.TeleportLevel(nil)
		client.server.BroadcastMessage(ColorYellow + client.entity.name + " has left the game!")
		client.server.RemoveClient(client)
		client.server.RemoveEntity(client.entity)
		atomic.AddInt32(&client.server.playerCount, -1)
	}

	event := EventClientDisconnect{client}
	client.server.FireEvent(EventTypeClientDisconnect, &event)
}

func (client *Client) Kick(reason string) {
	client.sendPacket(&packetDisconnect{
		packetTypeDisconnect,
		padString(reason),
	})

	client.Disconnect()
}

func (client *Client) Operator() bool {
	return client.operator
}

func (client *Client) SetOperator(value bool) {
	if client.loggedIn == 1 && value != client.operator {
		userType := byte(0x00)
		if value {
			userType = 0x64
		}

		client.sendPacket(&packetUpdateUserType{
			packetTypeUpdateUserType,
			userType,
		})
	}

	client.operator = value
}

func (client *Client) ClickDistance() float64 {
	return client.clickDistance
}

func (client *Client) SetClickDistance(value float64) {
	if client.loggedIn == 1 && client.HasExtension("ClickDistance") {
		client.sendPacket(&packetSetClickDistance{
			packetTypeSetClickDistance,
			int16(value * 32),
		})
	}

	client.clickDistance = value
}

func (client *Client) HeldBlock() BlockID {
	return client.heldBlock
}

func (client *Client) SetHeldBlock(block BlockID, lock bool) {
	if client.loggedIn == 1 && client.HasExtension("HeldBlock") {
		preventChange := byte(0)
		if lock {
			preventChange = 1
		}

		client.sendPacket(&packetHoldThis{
			packetTypeHoldThis,
			client.convertBlock(block),
			preventChange,
		})
	}
}

func (client *Client) SendMessage(message string) {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		client.sendPacket(&packetMessage{
			packetTypeMessage,
			0x00,
			padString(line),
		})
	}
}

func (client *Client) SetSpawn() {
	client.sendSpawn(client.entity)
}

func (client *Client) sendPacket(packet interface{}) {
	if client.connected == 0 {
		return
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, packet)
	_, err := buffer.WriteTo(client.conn)
	if err == io.EOF {
		client.Disconnect()
	}
}

func (client *Client) convertBlock(block BlockID) byte {
	if client.customBlockSupportLevel < 1 {
		return byte(FallbackBlock(block))
	}

	return byte(block)
}

func (client *Client) sendLevel(level *Level) {
	if client.loggedIn == 0 {
		return
	}

	client.sendPacket(&packetLevelInitialize{packetTypeLevelInitialize})

	var GZIPBuffer bytes.Buffer
	GZIPWriter := gzip.NewWriter(&GZIPBuffer)
	binary.Write(GZIPWriter, binary.BigEndian, int32(level.Volume()))
	for _, block := range level.blocks {
		GZIPWriter.Write([]byte{client.convertBlock(block)})
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

		packet := &packetLevelDataChunk{
			packetTypeLevelDataChunk,
			int16(size),
			[1024]byte{},
			byte(i * 100 / packets),
		}

		copy(packet.ChunkData[:], GZIPData[offset:offset+size])
		client.sendPacket(packet)
	}

	client.sendWeather(level.weather)
	client.sendEnvConfig(level.envConfig)

	client.sendPacket(&packetLevelFinalize{
		packetTypeLevelFinalize,
		int16(level.width), int16(level.height), int16(level.length),
	})
}

func (client *Client) sendSpawn(entity *Entity) {
	if client.loggedIn == 0 {
		return
	}

	id := entity.id
	if id == client.entity.id {
		id = 0xff
	}

	location := entity.location
	if client.HasExtension("ExtPlayerList") {
		client.sendPacket(&packetExtAddEntity2{
			packetTypeExtAddEntity2,
			id,
			padString(entity.DisplayName),
			padString(entity.SkinName),
			int16(location.X * 32),
			int16(location.Y * 32),
			int16(location.Z * 32),
			byte(location.Yaw * 256 / 360),
			byte(location.Pitch * 256 / 360),
		})
	} else {
		client.sendPacket(&packetSpawnPlayer{
			packetTypeSpawnPlayer,
			id,
			padString(entity.DisplayName),
			int16(location.X * 32),
			int16(location.Y * 32),
			int16(location.Z * 32),
			byte(location.Yaw * 256 / 360),
			byte(location.Pitch * 256 / 360),
		})
	}

	if entity.model != ModelHumanoid {
		client.sendChangeModel(entity)
	}
}

func (client *Client) sendDespawn(entity *Entity) {
	if client.loggedIn == 0 {
		return
	}

	id := entity.id
	if id == client.entity.id {
		id = 0xff
	}

	client.sendPacket(&packetDespawnPlayer{
		packetTypeDespawnPlayer,
		id,
	})
}

func (client *Client) sendTeleport(entity *Entity) {
	if client.loggedIn == 0 {
		return
	}

	id := entity.id
	if id == client.entity.id {
		id = 0xff
	}

	client.sendPacket(&packetPlayerTeleport{
		packetTypePlayerTeleport,
		id,
		int16(entity.location.X * 32),
		int16(entity.location.Y * 32),
		int16(entity.location.Z * 32),
		byte(entity.location.Yaw * 256 / 360),
		byte(entity.location.Pitch * 256 / 360),
	})
}

func (client *Client) sendBlockChange(x, y, z uint, block BlockID) {
	if client.loggedIn == 0 {
		return
	}

	client.sendPacket(&packetSetBlock{
		packetTypeSetBlock,
		int16(x), int16(y), int16(z),
		client.convertBlock(block),
	})
}

func (client *Client) sendCPE() {
	client.sendPacket(&packetExtInfo{
		packetTypeExtInfo,
		padString(ServerSoftware),
		int16(len(Extensions)),
	})

	for _, extension := range Extensions {
		client.sendPacket(&packetExtEntry{
			packetTypeExtEntry,
			padString(extension.Name),
			int32(extension.Version),
		})
	}
}

func (client *Client) sendAddPlayerList(entity *Entity) {
	if client.loggedIn == 0 || !client.HasExtension("ExtPlayerList") {
		return
	}

	id := entity.id
	if id == client.entity.id {
		id = 0xff
	}

	client.sendPacket(&packetExtAddPlayerName{
		packetTypeExtAddPlayerName,
		int16(id),
		padString(entity.name),
		padString(entity.listName),
		padString(entity.groupName),
		entity.groupRank,
	})
}

func (client *Client) sendRemovePlayerList(entity *Entity) {
	if client.loggedIn == 0 || !client.HasExtension("ExtPlayerList") {
		return
	}

	id := entity.id
	if id == client.entity.id {
		id = 0xff
	}

	client.sendPacket(&packetExtRemovePlayerName{
		packetTypeExtRemovePlayerName,
		int16(id),
	})
}

func (client *Client) sendChangeModel(entity *Entity) {
	if client.loggedIn == 0 || !client.HasExtension("ChangeModel") {
		return
	}

	id := entity.id
	if id == client.entity.id {
		id = 0xff
	}

	client.sendPacket(&packetChangeModel{
		packetTypeChangeModel,
		id,
		padString(entity.model),
	})
}

func (client *Client) sendWeather(weather WeatherType) {
	if client.loggedIn == 0 || !client.HasExtension("EnvWeatherType") {
		return
	}

	client.sendPacket(&packetEnvSetWeatherType{
		packetTypeEnvSetWeatherType,
		byte(weather),
	})
}

func (client *Client) sendTexturePack(texturePack string) {
	if client.loggedIn == 0 || !client.HasExtension("EnvMapAspect") {
		return
	}

	client.sendPacket(&packetSetMapEnvUrl{
		packetTypeSetMapEnvUrl,
		padString(texturePack),
	})
}

func (client *Client) sendEnvProp(id byte, value int) {
	client.sendPacket(&packetSetMapEnvProperty{
		packetTypeSetMapEnvProperty,
		id, int32(value),
	})
}

func (client *Client) sendEnvConfig(env EnvConfig) {
	if client.loggedIn == 0 || !client.HasExtension("EnvMapAspect") {
		return
	}

	client.sendEnvProp(0, int(client.convertBlock(env.SideBlock)))
	client.sendEnvProp(1, int(client.convertBlock(env.EdgeBlock)))
	client.sendEnvProp(2, int(env.EdgeHeight))
	client.sendEnvProp(3, int(env.CloudHeight))
	client.sendEnvProp(4, int(env.MaxViewDistance))
	client.sendEnvProp(5, int(256*env.CloudSpeed))
	client.sendEnvProp(6, int(256*env.WeatherSpeed))
	client.sendEnvProp(7, int(128*env.WeatherFade))
	client.sendEnvProp(9, env.SideOffset)

	if env.ExpFog {
		client.sendEnvProp(8, 1)
	} else {
		client.sendEnvProp(8, 0)
	}
}

func (client *Client) handle() {
	buffer := make([]byte, 256)
	atomic.StoreUint32(&client.connected, 1)
	for client.connected == 1 {
		buffer = buffer[:1]
		_, err := io.ReadFull(client.conn, buffer)
		if err != nil {
			client.Disconnect()
			return
		}

		id := buffer[0]
		var size uint
		switch id {
		case packetTypeIdentification:
			size = 131
		case packetTypeSetBlockClient:
			size = 9
		case packetTypePlayerTeleport:
			size = 10
		case packetTypeMessage:
			size = 66
		case packetTypeExtInfo:
			size = 67
		case packetTypeExtEntry:
			size = 69
		case packetTypeCustomBlockSupportLevel:
			size = 2
		case packetTypePlayerClicked:
			size = 15

		default:
			fmt.Printf("Invalid Packet: %d\n", id)
			continue
		}

		buffer = buffer[:size]
		_, err = io.ReadFull(client.conn, buffer[1:])
		if err != nil {
			client.Disconnect()
			return
		}

		reader := bytes.NewReader(buffer)
		switch id {
		case packetTypeIdentification:
			client.handleIdentification(reader)
		case packetTypeSetBlockClient:
			client.handleSetBlock(reader)
		case packetTypePlayerTeleport:
			client.handlePlayerTeleport(reader)
		case packetTypeMessage:
			client.handleMessage(reader)
		case packetTypeExtInfo:
			client.handleExtInfo(reader)
		case packetTypeExtEntry:
			client.handleExtEntry(reader)
		case packetTypeCustomBlockSupportLevel:
			client.handleCustomBlockSupportLevel(reader)
		case packetTypePlayerClicked:
			client.handlePlayerClicked(reader)
		}
	}
}

func (client *Client) login() {
	if client.loggedIn == 1 {
		return
	}

	for {
		count := client.server.playerCount
		if int(count) >= client.server.Config.MaxPlayers {
			client.Kick("Server full!")
			return
		}

		if atomic.CompareAndSwapInt32(&client.server.playerCount, count, count+1) {
			break
		}
	}

	if client.HasExtension("CustomBlocks") {
		client.sendPacket(&packetCustomBlockSupportLevel{
			packetTypeCustomBlockSupportLevel,
			1,
		})
	}

	userType := byte(0x00)
	if client.operator {
		userType = 0x64
	}

	client.sendPacket(&packetServerIdentification{
		packetTypeIdentification,
		0x07,
		padString(client.server.Config.Name),
		padString(client.server.Config.MOTD),
		userType,
	})

	client.entity = NewEntity(client.NickName, client.server, client)

	event := EventPlayerJoin{client.entity, false, ""}
	client.server.FireEvent(EventTypePlayerJoin, &event)
	if event.Cancel {
		client.Kick(event.CancelReason)
		return
	}

	atomic.StoreUint32(&client.loggedIn, 1)
	client.server.AddClient(client)
	client.server.BroadcastMessage(ColorYellow + client.entity.name + " has joined the game!")
	if client.server.AddEntity(client.entity) == 0xff {
		client.Kick("Server full!")
		return
	}

	client.server.ForEachEntity(func(entity *Entity) {
		if entity != client.entity {
			client.sendAddPlayerList(entity)
		}
	})

	if client.server.MainLevel != nil {
		client.entity.TeleportLevel(client.server.MainLevel)
	}

	client.pingTicker = time.NewTicker(2 * time.Second)
	go func() {
		for range client.pingTicker.C {
			client.sendPacket(&packetPing{packetTypePing})
		}
	}()
}

func (client *Client) verify(key []byte) bool {
	if len(key) != md5.Size {
		return false
	}

	data := make([]byte, len(client.server.salt))
	copy(data, client.server.salt[:])
	data = append(data, []byte(client.NickName)...)

	digest := md5.Sum(data)
	return bytes.Equal(digest[:], key)
}

func (client *Client) handleIdentification(reader io.Reader) {
	if client.loggedIn == 1 {
		return
	}

	packet := packetClientIdentification{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.ProtocolVersion != 0x07 {
		client.Kick("Wrong version!")
		return
	}

	client.NickName = trimString(packet.Name)
	if !IsValidName(client.NickName) {
		client.Kick("Invalid name!")
		return
	}

	key := trimString(packet.VerificationKey)
	if client.server.Config.Verify {
		if !client.verify([]byte(key)) {
			client.Kick("Login failed!")
			return
		}
	}

	if client.server.FindEntity(client.NickName) != nil {
		client.Kick("Already logged in!")
		return
	}

	if packet.Type == 0x42 {
		client.sendCPE()
	} else {
		client.login()
	}
}

func (client *Client) revertBlock(x, y, z uint) {
	client.sendBlockChange(x, y, z, client.entity.level.GetBlock(x, y, z))
}

func (client *Client) handleSetBlock(reader io.Reader) {
	if client.loggedIn == 0 {
		return
	}

	packet := packetSetBlockClient{}
	binary.Read(reader, binary.BigEndian, &packet)
	x, y, z := uint(packet.X), uint(packet.Y), uint(packet.Z)
	block := BlockID(packet.BlockType)

	dx := uint(client.entity.location.X) - x
	dy := uint(client.entity.location.Y) - y
	dz := uint(client.entity.location.Z) - z
	if math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) > client.clickDistance {
		client.SendMessage("You can't build that far away.")
		client.revertBlock(x, y, z)
		return
	}

	switch packet.Mode {
	case 0x00:
		event := &EventBlockBreak{
			client.entity,
			client.entity.level,
			client.entity.level.GetBlock(x, y, z),
			x, y, z,
			false,
		}
		client.server.FireEvent(EventTypeBlockBreak, &event)
		if event.Cancel {
			client.revertBlock(x, y, z)
			return
		}

		client.entity.level.SetBlock(x, y, z, BlockAir, true)

	case 0x01:
		if block > BlockMaxCPE || (client.customBlockSupportLevel < 1 && block > BlockMax) {
			client.SendMessage("Invalid block!")
			client.revertBlock(x, y, z)
			return
		}

		event := &EventBlockPlace{
			client.entity,
			client.entity.level,
			block,
			x, y, z,
			false,
		}
		client.server.FireEvent(EventTypeBlockPlace, &event)
		if event.Cancel {
			client.revertBlock(x, y, z)
			return
		}

		client.entity.level.SetBlock(x, y, z, block, true)
	}
}

func (client *Client) handlePlayerTeleport(reader io.Reader) {
	if client.loggedIn == 0 {
		return
	}

	packet := packetPlayerTeleport{}
	binary.Read(reader, binary.BigEndian, &packet)
	if client.HasExtension("HeldBlock") {
		client.heldBlock = BlockID(packet.PlayerID)
	} else if packet.PlayerID != 0xff {
		return
	}

	location := Location{
		float64(packet.X) / 32,
		float64(packet.Y) / 32,
		float64(packet.Z) / 32,
		float64(packet.Yaw) * 360 / 256,
		float64(packet.Pitch) * 360 / 256,
	}

	if location == client.entity.location {
		return
	}

	event := &EventEntityMove{client.entity, client.entity.location, location, false}
	client.server.FireEvent(EventTypeEntityMove, &event)
	if event.Cancel {
		client.sendTeleport(client.entity)
		return
	}

	client.entity.location = location
}

func (client *Client) handleMessage(reader io.Reader) {
	if client.loggedIn == 0 {
		return
	}

	packet := packetMessage{}
	binary.Read(reader, binary.BigEndian, &packet)

	client.message += trimString(packet.Message)
	if packet.PlayerID != 0x00 && client.HasExtension("LongerMessages") {
		return
	}

	message := client.message
	client.message = ""

	if !IsValidMessage(message) {
		client.SendMessage("Invalid message!")
		return
	}

	if message[0] == '/' {
		client.server.ExecuteCommand(client, message[1:])
	} else {
		client.server.BroadcastMessage(ColorDefault + "<" + client.NickName + "> " + ConvertColors(message))
	}
}

func (client *Client) handleExtInfo(reader io.Reader) {
	packet := packetExtInfo{}
	binary.Read(reader, binary.BigEndian, &packet)

	client.remainingExtensions = uint(packet.ExtensionCount)
	if client.remainingExtensions == 0 {
		client.login()
	}
}

func (client *Client) handleExtEntry(reader io.Reader) {
	packet := packetExtEntry{}
	binary.Read(reader, binary.BigEndian, &packet)

	for _, extension := range Extensions {
		if extension.Name == trimString(packet.ExtName) {
			if extension.Version == int(packet.Version) {
				client.extensions[extension.Name] = int(packet.Version)
				break
			}
		}
	}

	client.remainingExtensions--
	if client.remainingExtensions == 0 {
		client.login()
	}
}

func (client *Client) handleCustomBlockSupportLevel(reader io.Reader) {
	packet := packetCustomBlockSupportLevel{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.SupportLevel <= 1 {
		client.customBlockSupportLevel = packet.SupportLevel
	}
}

func (client *Client) handlePlayerClicked(reader io.Reader) {
	packet := packetPlayerClicked{}
	binary.Read(reader, binary.BigEndian, &packet)

	var target *Entity = nil
	if packet.TargetID != 0xff {
		target = client.server.FindEntityByID(packet.TargetID)
	}

	event := EventClientClick{
		client,
		packet.Button, packet.Action,
		float64(packet.Yaw) * 360 / 65536,
		float64(packet.Pitch) * 360 / 65536,
		target,
		uint(packet.BlockX), uint(packet.BlockY), uint(packet.BlockZ),
		packet.BlockFace,
	}
	client.server.FireEvent(EventTypeClientClick, &event)
}
