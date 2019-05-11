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
	var packet Packet
	packet.kick(reason)
	player.sendPacket(packet)

	player.Disconnect()
}

func (player *Player) Operator() bool {
	return player.operator
}

func (player *Player) SetOperator(value bool) {
	player.operator = value
	if player.state == stateGame && value != player.operator {
		var packet Packet
		packet.userType(player)
		player.sendPacket(packet)
	}
}

func (player *Player) ClickDistance() float64 {
	return player.clickDistance
}

func (player *Player) SetClickDistance(value float64) {
	player.clickDistance = value
	if player.state == stateGame && player.cpe[CpeClickDistance] {
		var packet Packet
		packet.clickDistance(player)
		player.sendPacket(packet)
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
	if player.state == stateGame && player.cpe[CpeHeldBlock] {
		var packet Packet
		packet.holdThis(player.convertBlock(block), lock)
		player.sendPacket(packet)
	}
}

func (player *Player) SetSelection(id int, label string, box AABB, color color.RGBA) {
	if player.state == stateGame && player.cpe[CpeSelectionCuboid] {
		var packet Packet
		packet.makeSelection(id, label, box, color)
		player.sendPacket(packet)
	}
}

func (player *Player) ResetSelection(id int) {
	if player.state == stateGame && player.cpe[CpeSelectionCuboid] {
		var packet Packet
		packet.removeSelection(id)
		player.sendPacket(packet)
	}
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

	var packet Packet
	for _, line := range WordWrap(message, 64) {
		packet.message(msgType, line)
	}

	player.sendPacket(packet)
}

func (player *Player) SetSpawn() {
	player.sendSpawn(player.Entity)
}

func (player *Player) sendPacket(packet Packet) {
	if player.state == stateClosed {
		return
	}

	_, err := packet.buf.WriteTo(player.conn)
	if err == io.EOF {
		player.Disconnect()
	}
}

func (player *Player) convertBlock(block BlockID) BlockID {
	if player.cpeBlockLevel < 1 {
		return FallbackBlock(block)
	}

	return block
}

func (player *Player) sendMOTD(level *Level) {
	motd := level.MOTD
	if len(motd) == 0 {
		motd = player.server.Config.MOTD
	}

	var packet Packet
	packet.motd(player, motd)
	player.sendPacket(packet)
}

func (player *Player) sendLevel(level *Level) {
	if player.state != stateGame {
		return
	}

	player.sendMOTD(level)

	var conv [BlockCountCPE]byte
	for i := 0; i < BlockCountCPE; i++ {
		conv[i] = byte(player.convertBlock(BlockID(i)))
	}

	stream := levelStream{player: player}
	if player.cpe[CpeFastMap] {
		var packet Packet
		packet.levelInitializeExt(level.Volume())
		player.sendPacket(packet)

		writer, _ := flate.NewWriter(&stream, -1)
		for i, block := range level.Blocks {
			stream.percent = byte(i * 100 / len(level.Blocks))
			writer.Write([]byte{conv[block]})
		}
		writer.Close()
	} else {
		var packet Packet
		packet.levelInitialize()
		player.sendPacket(packet)

		writer := gzip.NewWriter(&stream)
		binary.Write(writer, binary.BigEndian, int32(level.Volume()))
		for i, block := range level.Blocks {
			stream.percent = byte(i * 100 / len(level.Blocks))
			writer.Write([]byte{conv[block]})
		}
		writer.Close()
	}
	stream.Close()

	player.sendWeather(level)
	player.sendTexturePack(level)
	player.sendEnvConfig(level, EnvPropAll)
	player.sendHackConfig(level)

	var packet Packet
	packet.levelFinalize(level.width, level.height, level.length)
	player.sendPacket(packet)
}

func (player *Player) sendSpawn(entity *Entity) {
	if player.state != stateGame {
		return
	}

	var packet Packet
	self := entity.id == player.id
	extPos := player.cpe[CpeExtEntityPositions]
	if player.cpe[CpeExtPlayerList] {
		packet.extAddEntity2(entity, self, extPos)
	} else {
		packet.addEntity(entity, self, extPos)
	}

	player.sendPacket(packet)

	if entity.Model != ModelHumanoid {
		player.sendChangeModel(entity)
	}

	player.sendEntityProps(entity, EntityPropAll)
}

func (player *Player) sendDespawn(entity *Entity) {
	if player.state == stateGame {
		var packet Packet
		packet.removeEntity(entity, entity.id == player.id)
		player.sendPacket(packet)
	}
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
	if player.state == stateGame {
		var packet Packet
		extPos := player.cpe[CpeExtEntityPositions]
		packet.teleport(entity, entity.id == player.id, extPos)
		player.sendPacket(packet)
	}
}

func (player *Player) sendBlockChange(x, y, z uint, block BlockID) {
	if player.state == stateGame {
		var packet Packet
		packet.setBlock(x, y, z, player.convertBlock(block))
		player.sendPacket(packet)
	}
}

func (player *Player) sendCPE() {
	var packet Packet
	packet.extInfo()
	for _, extension := range Extensions {
		packet.extEntry(&extension)
	}

	player.sendPacket(packet)
}

func (player *Player) sendAddPlayerList(entity *Entity) {
	if player.state == stateGame && player.cpe[CpeExtPlayerList] {
		var packet Packet
		packet.extAddPlayerName(entity, entity.id == player.id)
		player.sendPacket(packet)
	}
}

func (player *Player) sendRemovePlayerList(entity *Entity) {
	if player.state == stateGame && player.cpe[CpeExtPlayerList] {
		var packet Packet
		packet.extRemovePlayerName(entity, entity.id == player.id)
		player.sendPacket(packet)
	}
}

func (player *Player) sendChangeModel(entity *Entity) {
	if player.state == stateGame && player.cpe[CpeChangeModel] {
		var packet Packet
		packet.changeModel(entity, entity.id == player.id)
		player.sendPacket(packet)
	}
}

func (player *Player) sendWeather(level *Level) {
	if player.state == stateGame && player.cpe[CpeEnvWeatherType] {
		var packet Packet
		packet.envWeatherType(level)
		player.sendPacket(packet)
	}
}

func (player *Player) sendTexturePack(level *Level) {
	if player.state == stateGame && player.cpe[CpeEnvMapAspect] {
		var packet Packet
		packet.mapEnvUrl(level)
		player.sendPacket(packet)
	}
}

func (player *Player) sendEnvConfig(level *Level, mask uint32) {
	if player.state != stateGame || !player.cpe[CpeEnvMapAspect] {
		return
	}

	var packet Packet
	config := level.EnvConfig
	if mask&EnvPropSideBlock != 0 {
		packet.mapEnvProperty(0, int32(player.convertBlock(config.SideBlock)))
	}
	if mask&EnvPropEdgeBlock != 0 {
		packet.mapEnvProperty(1, int32(player.convertBlock(config.EdgeBlock)))
	}
	if mask&EnvPropEdgeHeight != 0 {
		packet.mapEnvProperty(2, int32(config.EdgeHeight))
	}
	if mask&EnvPropCloudHeight != 0 {
		packet.mapEnvProperty(3, int32(config.CloudHeight))
	}
	if mask&EnvPropMaxViewDistance != 0 {
		packet.mapEnvProperty(4, int32(config.MaxViewDistance))
	}
	if mask&EnvPropCloudSpeed != 0 {
		packet.mapEnvProperty(5, int32(256*config.CloudSpeed))
	}
	if mask&EnvPropWeatherSpeed != 0 {
		packet.mapEnvProperty(6, int32(256*config.WeatherSpeed))
	}
	if mask&EnvPropWeatherFade != 0 {
		packet.mapEnvProperty(7, int32(128*config.WeatherFade))
	}
	if mask&EnvPropExpFog != 0 {
		if config.ExpFog {
			packet.mapEnvProperty(8, 1)
		} else {
			packet.mapEnvProperty(8, 0)
		}
	}
	if mask&EnvPropSideOffset != 0 {
		packet.mapEnvProperty(9, int32(config.SideOffset))
	}

	player.sendPacket(packet)
}

func (player *Player) sendEntityProps(entity *Entity, mask uint32) {
	if player.state != stateGame || !player.cpe[CpeEntityProperty] {
		return
	}

	var packet Packet
	props := entity.Props
	self := entity.id == player.id
	if mask&EntityPropRotX != 0 {
		packet.entityProperty(entity, self, 0, int32(props.RotX))
	}
	if mask&EntityPropRotY != 0 {
		packet.entityProperty(entity, self, 1, int32(props.RotY))
	}
	if mask&EntityPropRotZ != 0 {
		packet.entityProperty(entity, self, 2, int32(props.RotZ))
	}
	if mask&EntityPropScaleX != 0 {
		packet.entityProperty(entity, self, 3, int32(1000*props.ScaleX))
	}
	if mask&EntityPropScaleY != 0 {
		packet.entityProperty(entity, self, 4, int32(1000*props.ScaleY))
	}
	if mask&EntityPropScaleZ != 0 {
		packet.entityProperty(entity, self, 5, int32(1000*props.ScaleZ))
	}

	player.sendPacket(packet)
}

func (player *Player) sendHackConfig(level *Level) {
	if player.state == stateGame && player.cpe[CpeHackControl] {
		var packet Packet
		packet.hackControl(&level.HackConfig)
		player.sendPacket(packet)
	}
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
				if player.cpe[CpeExtEntityPositions] {
					size = 16
				} else {
					size = 10
				}
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
			player.handleTeleport(reader)
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
		var packet Packet
		packet.ping()
		for range player.pingTicker.C {
			player.sendPacket(packet)
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
	packet := struct {
		PacketID        byte
		ProtocolVersion byte
		Name            [64]byte
		VerificationKey [64]byte
		Type            byte
	}{}
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
	player.ListName = player.name

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
	packet := struct {
		PacketID  byte
		X, Y, Z   int16
		Mode      byte
		BlockType byte
	}{}
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

func (player *Player) handleTeleport(reader io.Reader) {
	packet0 := struct{ PacketID, PlayerID byte }{}
	binary.Read(reader, binary.BigEndian, &packet0)

	location := Location{}
	if player.cpe[CpeExtEntityPositions] {
		packet1 := struct{ X, Y, Z int32 }{}
		binary.Read(reader, binary.BigEndian, &packet1)
		location.X = float64(packet1.X) / 32
		location.Y = float64(packet1.Y) / 32
		location.Z = float64(packet1.Z) / 32
	} else {
		packet1 := struct{ X, Y, Z int16 }{}
		binary.Read(reader, binary.BigEndian, &packet1)
		location.X = float64(packet1.X) / 32
		location.Y = float64(packet1.Y) / 32
		location.Z = float64(packet1.Z) / 32
	}

	packet2 := struct{ Yaw, Pitch byte }{}
	binary.Read(reader, binary.BigEndian, &packet2)
	location.Yaw = float64(packet2.Yaw) * 360 / 256
	location.Pitch = float64(packet2.Pitch) * 360 / 256

	if player.cpe[CpeHeldBlock] {
		player.heldBlock = BlockID(packet0.PlayerID)
	} else if packet0.PlayerID != 0xff {
		return
	}

	if player.level == nil || location == player.location {
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
	packet := struct {
		PacketID byte
		PlayerID byte
		Message  [64]byte
	}{}
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
	packet := struct {
		PacketID       byte
		AppName        [64]byte
		ExtensionCount int16
	}{}
	binary.Read(reader, binary.BigEndian, &packet)

	player.remExtensions = uint(packet.ExtensionCount)
	if player.remExtensions == 0 {
		player.login()
	}
}

func (player *Player) handleExtEntry(reader io.Reader) {
	packet := struct {
		PacketID byte
		ExtName  [64]byte
		Version  int32
	}{}
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
			var packet Packet
			packet.customBlockSupportLevel(1)
			player.sendPacket(packet)
		} else {
			player.login()
		}
	}
}

func (player *Player) handleCustomBlockSupportLevel(reader io.Reader) {
	packet := struct{ PacketID, SupportLevel byte }{}
	binary.Read(reader, binary.BigEndian, &packet)
	if packet.SupportLevel <= 1 {
		player.cpeBlockLevel = packet.SupportLevel
	}

	player.login()
}

func (player *Player) handlePlayerClicked(reader io.Reader) {
	packet := struct {
		PacketID               byte
		Button, Action         byte
		Yaw, Pitch             int16
		TargetID               byte
		BlockX, BlockY, BlockZ int16
		BlockFace              byte
	}{}
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
	packet := struct {
		PacketID  byte
		Direction byte
		Data      int16
	}{}
	binary.Read(reader, binary.BigEndian, &packet)

	switch packet.Direction {
	case 0:
		var response Packet
		response.twoWayPing(0, packet.Data)
		player.sendPacket(response)
	}
}
