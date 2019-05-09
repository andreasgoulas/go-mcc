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
	"compress/flate"
	"compress/gzip"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	stateClosed = 0
	stateLogin  = 1
	stateGame   = 2
)

type Player struct {
	*Entity

	Nickname string

	conn  net.Conn
	state uint32

	operator       bool
	permGroupsLock sync.RWMutex
	permGroups     []*PermissionGroup

	cpe           [CpeCount]bool
	remExtensions uint
	message       string
	cpeBlockLevel byte
	clickDistance float64
	heldBlock     BlockID

	pingTicker *time.Ticker
}

func NewPlayer(conn net.Conn, server *Server) *Player {
	return &Player{
		Entity:        NewEntity("", server),
		conn:          conn,
		state:         stateClosed,
		clickDistance: 5.0,
	}
}

func (player *Player) AddPermissionGroup(group *PermissionGroup) {
	player.permGroupsLock.Lock()
	player.permGroups = append(player.permGroups, group)
	player.permGroupsLock.Unlock()
}

func (player *Player) RemovePermissionGroup(group *PermissionGroup) {
	player.permGroupsLock.RLock()
	defer player.permGroupsLock.RUnlock()

	index := -1
	for i, g := range player.permGroups {
		if g == group {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	player.permGroups[index] = player.permGroups[len(player.permGroups)-1]
	player.permGroups[len(player.permGroups)-1] = nil
	player.permGroups = player.permGroups[:len(player.permGroups)-1]
}

func (player *Player) HasPermission(permission string) bool {
	player.permGroupsLock.RLock()
	defer player.permGroupsLock.RUnlock()

	for _, group := range player.permGroups {
		if group.HasPermission(permission) {
			return true
		}
	}

	return false
}

func (player *Player) HasExtension(extension uint) bool {
	return player.cpe[extension]
}

func (player *Player) RemoteAddr() string {
	addr := player.conn.RemoteAddr()
	host, _, _ := net.SplitHostPort(addr.String())
	return host
}

func (player *Player) Disconnect() {
	if player.state == stateClosed {
		return
	}

	if player.pingTicker != nil {
		player.pingTicker.Stop()
	}

	loggedIn := player.state == stateGame
	atomic.StoreUint32(&player.state, stateClosed)
	player.conn.Close()

	if loggedIn {
		event := EventPlayerQuit{player}
		player.server.FireEvent(EventTypePlayerQuit, &event)

		player.TeleportLevel(nil)
		player.server.BroadcastMessage(ColorYellow + player.name + " has left the game!")
		player.server.RemoveEntity(player.Entity)
		player.server.RemovePlayer(player)
		atomic.AddInt32(&player.server.playerCount, -1)
	}
}

func (player *Player) Kick(reason string) {
	player.sendPacket(&packetDisconnect{
		packetTypeDisconnect,
		padString(reason),
	})

	player.Disconnect()
}

func (player *Player) Operator() bool {
	return player.operator
}

func (player *Player) SetOperator(value bool) {
	player.operator = value
	if player.state == stateGame && value != player.operator {
		userType := byte(0x00)
		if value {
			userType = 0x64
		}

		player.sendPacket(&packetUpdateUserType{
			packetTypeUpdateUserType,
			userType,
		})
	}
}

func (player *Player) ClickDistance() float64 {
	return player.clickDistance
}

func (player *Player) SetClickDistance(value float64) {
	player.clickDistance = value
	if player.state == stateGame && player.cpe[CpeClickDistance] {
		player.sendPacket(&packetSetClickDistance{
			packetTypeSetClickDistance,
			int16(value * 32),
		})
	}
}

func (player *Player) CanReach(x, y, z uint) bool {
	loc := player.location
	dx := math.Min(math.Abs(loc.X-float64(x)), math.Abs(loc.X-float64(x+1)))
	dy := math.Min(math.Abs(loc.Y-float64(y)), math.Abs(loc.Y-float64(y+1)))
	dz := math.Min(math.Abs(loc.Z-float64(z)), math.Abs(loc.Z-float64(z+1)))
	return dx*dx+dy*dy+dz*dz <= player.clickDistance*player.clickDistance
}

func (player *Player) HeldBlock() BlockID {
	return player.heldBlock
}

func (player *Player) SetHeldBlock(block BlockID, lock bool) {
	if player.state != stateGame || !player.cpe[CpeHeldBlock] {
		return
	}

	preventChange := byte(0)
	if lock {
		preventChange = 1
	}

	player.sendPacket(&packetHoldThis{
		packetTypeHoldThis,
		player.convertBlock(block),
		preventChange,
	})
}

func (player *Player) SetSelection(id int, label string, box AABB, color color.RGBA) {
	if player.state != stateGame || !player.cpe[CpeSelectionCuboid] {
		return
	}

	player.sendPacket(&packetMakeSelection{
		packetTypeMakeSelection,
		byte(id),
		padString(label),
		int16(box.Min.X), int16(box.Min.Y), int16(box.Min.Z),
		int16(box.Max.X), int16(box.Max.Y), int16(box.Max.Z),
		int16(color.R), int16(color.G), int16(color.B), int16(color.A),
	})
}

func (player *Player) ResetSelection(id int) {
	if player.state != stateGame || !player.cpe[CpeSelectionCuboid] {
		return
	}

	player.sendPacket(&packetRemoveSelection{
		packetTypeRemoveSelection,
		byte(id),
	})
}

func (player *Player) SendMessage(message string) {
	player.SendMessageExt(MessageChat, message)
}

func (player *Player) SendMessageExt(msgType int, message string) {
	if msgType != MessageChat && !player.cpe[CpeMessageTypes] {
		if msgType == MessageAnnouncement {
			msgType = MessageChat
		} else {
			return
		}
	}

	for _, line := range WordWrap(message, 64) {
		player.sendPacket(&packetMessage{
			packetTypeMessage,
			byte(msgType),
			padString(line),
		})
	}
}

func (player *Player) SetSpawn() {
	player.sendSpawn(player.Entity)
}

func (player *Player) sendPacket(packet interface{}) {
	if player.state == stateClosed {
		return
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, packet)
	_, err := buffer.WriteTo(player.conn)
	if err == io.EOF {
		player.Disconnect()
	}
}

func (player *Player) convertBlock(block BlockID) byte {
	if player.cpeBlockLevel < 1 {
		return byte(FallbackBlock(block))
	}

	return byte(block)
}

func (player *Player) sendMOTD(level *Level) {
	motd := level.MOTD
	if len(motd) == 0 {
		motd = player.server.Config.MOTD
	}

	userType := byte(0x00)
	if player.operator {
		userType = 0x64
	}

	player.sendPacket(&packetServerIdentification{
		packetTypeIdentification,
		0x07,
		padString(player.server.Config.Name),
		padString(motd),
		userType,
	})
}

func (player *Player) sendLevel(level *Level) {
	if player.state != stateGame {
		return
	}

	player.sendMOTD(level)

	var buffer bytes.Buffer
	if player.cpe[CpeFastMap] {
		player.sendPacket(&packetLevelInitializeExt{
			packetTypeLevelInitialize,
			int32(level.Volume()),
		})

		writer, _ := flate.NewWriter(&buffer, -1)
		for _, block := range level.blocks {
			writer.Write([]byte{player.convertBlock(block)})
		}
		writer.Close()
	} else {
		player.sendPacket(&packetLevelInitialize{packetTypeLevelInitialize})

		writer := gzip.NewWriter(&buffer)
		binary.Write(writer, binary.BigEndian, int32(level.Volume()))
		for _, block := range level.blocks {
			writer.Write([]byte{player.convertBlock(block)})
		}
		writer.Close()
	}

	data := buffer.Bytes()
	packets := int(math.Ceil(float64(len(data)) / 1024))
	for i := 0; i < packets; i++ {
		offset := 1024 * i
		size := len(data) - offset
		if size > 1024 {
			size = 1024
		}

		packet := &packetLevelDataChunk{
			packetTypeLevelDataChunk,
			int16(size),
			[1024]byte{},
			byte(i * 100 / packets),
		}

		copy(packet.ChunkData[:], data[offset:offset+size])
		player.sendPacket(packet)
	}

	player.sendWeather(level.weather)
	player.sendEnvConfig(level.envConfig)
	player.sendHackConfig(level.hackConfig)

	player.sendPacket(&packetLevelFinalize{
		packetTypeLevelFinalize,
		int16(level.width), int16(level.height), int16(level.length),
	})
}

func (player *Player) sendSpawn(entity *Entity) {
	if player.state != stateGame {
		return
	}

	id := entity.id
	if id == player.id {
		id = 0xff
	}

	location := entity.location
	if player.cpe[CpeExtPlayerList] {
		player.sendPacket(&packetExtAddEntity2{
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
		player.sendPacket(&packetSpawnPlayer{
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
		player.sendChangeModel(entity)
	}
}

func (player *Player) sendDespawn(entity *Entity) {
	if player.state != stateGame {
		return
	}

	id := entity.id
	if id == player.id {
		id = 0xff
	}

	player.sendPacket(&packetDespawnPlayer{
		packetTypeDespawnPlayer,
		id,
	})
}

func (player *Player) spawnLevel(level *Level) {
	player.sendLevel(level)
	player.sendSpawn(player.Entity)
	level.ForEachEntity(func(other *Entity) {
		player.sendSpawn(other)
	})
}

func (player *Player) despawnLevel(level *Level) {
	player.sendDespawn(player.Entity)
	level.ForEachEntity(func(other *Entity) {
		player.sendDespawn(other)
	})
}

func (player *Player) sendTeleport(entity *Entity) {
	if player.state != stateGame {
		return
	}

	id := entity.id
	if id == player.id {
		id = 0xff
	}

	player.sendPacket(&packetPlayerTeleport{
		packetTypePlayerTeleport,
		id,
		int16(entity.location.X * 32),
		int16(entity.location.Y * 32),
		int16(entity.location.Z * 32),
		byte(entity.location.Yaw * 256 / 360),
		byte(entity.location.Pitch * 256 / 360),
	})
}

func (player *Player) sendBlockChange(x, y, z uint, block BlockID) {
	if player.state != stateGame {
		return
	}

	player.sendPacket(&packetSetBlock{
		packetTypeSetBlock,
		int16(x), int16(y), int16(z),
		player.convertBlock(block),
	})
}

func (player *Player) sendCPE() {
	player.sendPacket(&packetExtInfo{
		packetTypeExtInfo,
		padString(ServerSoftware),
		int16(len(Extensions)),
	})

	for _, extension := range Extensions {
		player.sendPacket(&packetExtEntry{
			packetTypeExtEntry,
			padString(extension.Name),
			int32(extension.Version),
		})
	}
}

func (player *Player) sendAddPlayerList(entity *Entity) {
	if player.state != stateGame || !player.cpe[CpeExtPlayerList] {
		return
	}

	id := entity.id
	if id == player.id {
		id = 0xff
	}

	player.sendPacket(&packetExtAddPlayerName{
		packetTypeExtAddPlayerName,
		int16(id),
		padString(entity.name),
		padString(entity.listName),
		padString(entity.groupName),
		entity.groupRank,
	})
}

func (player *Player) sendRemovePlayerList(entity *Entity) {
	if player.state != stateGame || !player.cpe[CpeExtPlayerList] {
		return
	}

	id := entity.id
	if id == player.id {
		id = 0xff
	}

	player.sendPacket(&packetExtRemovePlayerName{
		packetTypeExtRemovePlayerName,
		int16(id),
	})
}

func (player *Player) sendChangeModel(entity *Entity) {
	if player.state != stateGame || !player.cpe[CpeChangeModel] {
		return
	}

	id := entity.id
	if id == player.id {
		id = 0xff
	}

	player.sendPacket(&packetChangeModel{
		packetTypeChangeModel,
		id,
		padString(entity.model),
	})
}

func (player *Player) sendWeather(weather WeatherType) {
	if player.state != stateGame || !player.cpe[CpeEnvWeatherType] {
		return
	}

	player.sendPacket(&packetEnvSetWeatherType{
		packetTypeEnvSetWeatherType,
		byte(weather),
	})
}

func (player *Player) sendTexturePack(texturePack string) {
	if player.state != stateGame || !player.cpe[CpeEnvMapAspect] {
		return
	}

	player.sendPacket(&packetSetMapEnvUrl{
		packetTypeSetMapEnvUrl,
		padString(texturePack),
	})
}

func (player *Player) sendEnvProp(id byte, value int) {
	player.sendPacket(&packetSetMapEnvProperty{
		packetTypeSetMapEnvProperty,
		id, int32(value),
	})
}

func (player *Player) sendEnvConfig(env EnvConfig) {
	if player.state != stateGame || !player.cpe[CpeEnvMapAspect] {
		return
	}

	player.sendEnvProp(0, int(player.convertBlock(env.SideBlock)))
	player.sendEnvProp(1, int(player.convertBlock(env.EdgeBlock)))
	player.sendEnvProp(2, int(env.EdgeHeight))
	player.sendEnvProp(3, int(env.CloudHeight))
	player.sendEnvProp(4, int(env.MaxViewDistance))
	player.sendEnvProp(5, int(256*env.CloudSpeed))
	player.sendEnvProp(6, int(256*env.WeatherSpeed))
	player.sendEnvProp(7, int(128*env.WeatherFade))
	player.sendEnvProp(9, env.SideOffset)

	if env.ExpFog {
		player.sendEnvProp(8, 1)
	} else {
		player.sendEnvProp(8, 0)
	}
}

func (player *Player) sendHackConfig(hackConfig HackConfig) {
	if player.state != stateGame || !player.cpe[CpeHackControl] {
		return
	}

	packet := &packetHackControl{
		packetTypeHackControl,
		0, 0, 0, 0, 0,
		int16(hackConfig.JumpHeight),
	}

	if hackConfig.Flying {
		packet.Flying = 1
	}
	if hackConfig.NoClip {
		packet.NoClip = 1
	}
	if hackConfig.Speeding {
		packet.Speeding = 1
	}
	if hackConfig.SpawnControl {
		packet.SpawnControl = 1
	}
	if hackConfig.ThirdPersonView {
		packet.ThirdPersonView = 1
	}

	player.sendPacket(packet)
}

func (player *Player) handle() {
	buffer := make([]byte, 256)
	atomic.StoreUint32(&player.state, stateLogin)
	for player.state != stateClosed {
		buffer = buffer[:1]
		_, err := io.ReadFull(player.conn, buffer)
		if err != nil {
			player.Disconnect()
			return
		}

		id := buffer[0]
		var size uint
		switch player.state {
		case stateLogin:
			switch id {
			case packetTypeIdentification:
				size = 131
			case packetTypeExtInfo:
				size = 67
			case packetTypeExtEntry:
				size = 69
			case packetTypeCustomBlockSupportLevel:
				size = 2
			}

		case stateGame:
			switch id {
			case packetTypeSetBlockClient:
				size = 9
			case packetTypePlayerTeleport:
				size = 10
			case packetTypeMessage:
				size = 66
			case packetTypePlayerClicked:
				size = 15
			case packetTypeTwoWayPing:
				size = 4
			}
		}

		if size == 0 {
			player.Kick("Invalid Packet")
			break
		}

		buffer = buffer[:size]
		_, err = io.ReadFull(player.conn, buffer[1:])
		if err != nil {
			player.Disconnect()
			return
		}

		reader := bytes.NewReader(buffer)
		switch id {
		case packetTypeIdentification:
			player.handleIdentification(reader)
		case packetTypeSetBlockClient:
			player.handleSetBlock(reader)
		case packetTypePlayerTeleport:
			player.handlePlayerTeleport(reader)
		case packetTypeMessage:
			player.handleMessage(reader)
		case packetTypeExtInfo:
			player.handleExtInfo(reader)
		case packetTypeExtEntry:
			player.handleExtEntry(reader)
		case packetTypeCustomBlockSupportLevel:
			player.handleCustomBlockSupportLevel(reader)
		case packetTypePlayerClicked:
			player.handlePlayerClicked(reader)
		case packetTypeTwoWayPing:
			player.handleTwoWayPing(reader)
		}
	}
}

func (player *Player) login() {
	if player.state != stateLogin {
		return
	}

	event := EventPlayerLogin{player, false, ""}
	player.server.FireEvent(EventTypePlayerLogin, &event)
	if event.Cancel {
		player.Kick(event.CancelReason)
		return
	}

	for {
		count := player.server.playerCount
		if int(count) >= player.server.Config.MaxPlayers {
			player.Kick("Server full!")
			return
		}

		if atomic.CompareAndSwapInt32(&player.server.playerCount, count, count+1) {
			break
		}
	}

	joinEvent := EventPlayerJoin{player}
	player.server.FireEvent(EventTypePlayerJoin, &joinEvent)

	if !player.server.AddEntity(player.Entity) {
		player.Kick("Server full!")
		return
	}

	player.Entity.player = player

	atomic.StoreUint32(&player.state, stateGame)
	player.server.AddPlayer(player)
	player.server.ForEachEntity(func(entity *Entity) {
		if entity != player.Entity {
			player.sendAddPlayerList(entity)
		}
	})

	player.server.BroadcastMessage(ColorYellow + player.name + " has joined the game!")

	if player.server.MainLevel != nil {
		player.TeleportLevel(player.server.MainLevel)
	}

	player.pingTicker = time.NewTicker(2 * time.Second)
	go func() {
		for range player.pingTicker.C {
			player.sendPacket(&packetPing{packetTypePing})
		}
	}()
}

func (player *Player) verify(key []byte) bool {
	if len(key) != md5.Size {
		return false
	}

	data := make([]byte, len(player.server.salt))
	copy(data, player.server.salt[:])
	data = append(data, []byte(player.name)...)

	digest := md5.Sum(data)
	return bytes.Equal(digest[:], key)
}

func (player *Player) handleIdentification(reader io.Reader) {
	packet := packetClientIdentification{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.ProtocolVersion != 0x07 {
		player.Kick("Wrong version!")
		return
	}

	player.name = trimString(packet.Name)
	if !IsValidName(player.name) {
		player.Kick("Invalid name!")
		return
	}

	player.Nickname = player.name
	player.DisplayName = player.name
	player.SkinName = player.name
	player.listName = player.name

	event := EventPlayerPreLogin{player, false, ""}
	player.server.FireEvent(EventTypePlayerPreLogin, &event)
	if event.Cancel {
		player.Kick(event.CancelReason)
		return
	}

	key := trimString(packet.VerificationKey)
	if player.server.Config.Verify {
		if !player.verify([]byte(key)) {
			player.Kick("Login failed!")
			return
		}
	}

	if player.server.FindEntity(player.name) != nil {
		player.Kick("Already logged in!")
		return
	}

	if packet.Type == 0x42 {
		player.sendCPE()
	} else {
		player.login()
	}
}

func (player *Player) revertBlock(x, y, z uint) {
	player.sendBlockChange(x, y, z, player.level.GetBlock(x, y, z))
}

func (player *Player) handleSetBlock(reader io.Reader) {
	packet := packetSetBlockClient{}
	binary.Read(reader, binary.BigEndian, &packet)
	x, y, z := uint(packet.X), uint(packet.Y), uint(packet.Z)
	block := BlockID(packet.BlockType)

	level := player.level
	if x >= level.width || y >= level.height || z >= level.length {
		return
	}

	if !player.CanReach(x, y, z) {
		player.SendMessage("You can't build that far away.")
		player.revertBlock(x, y, z)
		return
	}

	switch packet.Mode {
	case 0x00:
		event := &EventBlockBreak{
			player,
			level,
			level.GetBlock(x, y, z),
			x, y, z,
			false,
		}
		player.server.FireEvent(EventTypeBlockBreak, &event)
		if event.Cancel {
			player.revertBlock(x, y, z)
			return
		}

		level.SetBlock(x, y, z, BlockAir, true)

	case 0x01:
		if block > BlockMaxCPE || (player.cpeBlockLevel < 1 && block > BlockMax) {
			player.SendMessage("Invalid block!")
			player.revertBlock(x, y, z)
			return
		}

		event := &EventBlockPlace{
			player,
			level,
			block,
			x, y, z,
			false,
		}
		player.server.FireEvent(EventTypeBlockPlace, &event)
		if event.Cancel {
			player.revertBlock(x, y, z)
			return
		}

		level.SetBlock(x, y, z, block, true)
	}
}

func (player *Player) handlePlayerTeleport(reader io.Reader) {
	packet := packetPlayerTeleport{}
	binary.Read(reader, binary.BigEndian, &packet)

	if player.level == nil {
		return
	}

	if player.cpe[CpeHeldBlock] {
		player.heldBlock = BlockID(packet.PlayerID)
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

	if location == player.location {
		return
	}

	event := &EventEntityMove{player.Entity, player.location, location, false}
	player.server.FireEvent(EventTypeEntityMove, &event)
	if event.Cancel {
		player.sendTeleport(player.Entity)
		return
	}

	player.location = location
}

func (player *Player) handleMessage(reader io.Reader) {
	packet := packetMessage{}
	binary.Read(reader, binary.BigEndian, &packet)

	player.message += trimString(packet.Message)
	if packet.PlayerID != 0x00 && player.cpe[CpeLongerMessages] {
		return
	}

	message := player.message
	player.message = ""

	if !IsValidMessage(message) {
		player.SendMessage("Invalid message!")
		return
	}

	if message[0] == '/' {
		player.server.ExecuteCommand(player, message[1:])
	} else {
		player.server.playersLock.RLock()
		players := make([]*Player, len(player.server.players))
		copy(players, player.server.players)
		player.server.playersLock.RUnlock()

		event := EventPlayerChat{
			player,
			players,
			ConvertColors(message),
			"%s: &f%s",
			false,
		}
		player.server.FireEvent(EventTypePlayerChat, &event)
		if event.Cancel {
			return
		}

		message = fmt.Sprintf(event.Format, player.Nickname, event.Message)

		log.Printf(message)
		for _, player := range event.Targets {
			player.SendMessage(message)
		}
	}
}

func (player *Player) handleExtInfo(reader io.Reader) {
	packet := packetExtInfo{}
	binary.Read(reader, binary.BigEndian, &packet)

	player.remExtensions = uint(packet.ExtensionCount)
	if player.remExtensions == 0 {
		player.login()
	}
}

func (player *Player) handleExtEntry(reader io.Reader) {
	packet := packetExtEntry{}
	binary.Read(reader, binary.BigEndian, &packet)

	for i, extension := range Extensions {
		if extension.Name == trimString(packet.ExtName) {
			if extension.Version == int(packet.Version) {
				player.cpe[i] = true
				break
			}
		}
	}

	player.remExtensions--
	if player.remExtensions == 0 {
		if player.cpe[CpeCustomBlocks] {
			player.sendPacket(&packetCustomBlockSupportLevel{
				packetTypeCustomBlockSupportLevel,
				1,
			})
		} else {
			player.login()
		}
	}
}

func (player *Player) handleCustomBlockSupportLevel(reader io.Reader) {
	packet := packetCustomBlockSupportLevel{}
	binary.Read(reader, binary.BigEndian, &packet)

	if packet.SupportLevel <= 1 {
		player.cpeBlockLevel = packet.SupportLevel
	}

	player.login()
}

func (player *Player) handlePlayerClicked(reader io.Reader) {
	packet := packetPlayerClicked{}
	binary.Read(reader, binary.BigEndian, &packet)

	var target *Entity = nil
	if packet.TargetID != 0xff {
		target = player.server.FindEntityByID(packet.TargetID)
	}

	event := EventPlayerClick{
		player,
		packet.Button, packet.Action,
		float64(packet.Yaw) * 360 / 65536,
		float64(packet.Pitch) * 360 / 65536,
		target,
		uint(packet.BlockX), uint(packet.BlockY), uint(packet.BlockZ),
		packet.BlockFace,
	}
	player.server.FireEvent(EventTypePlayerClick, &event)
}

func (player *Player) handleTwoWayPing(reader io.Reader) {
	packet := packetTwoWayPing{}
	binary.Read(reader, binary.BigEndian, &packet)

	switch packet.Direction {
	case 0:
		player.sendPacket(&packetTwoWayPing{
			packetTypeTwoWayPing,
			0, packet.Data,
		})
	}
}
