// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

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
	"sync/atomic"
	"time"
)

const (
	stateClosed = 0
	stateLogin  = 1
	stateGame   = 2
)

// Player represents a game client.
type Player struct {
	*Entity

	Nickname string
	Rank     *Rank

	conn  net.Conn
	state uint32

	cpe           [CpeCount]bool
	remExtensions int
	message       string
	maxBlockID    byte
	cpeBlockLevel byte
	heldBlock     byte

	pingTicker *time.Ticker
	pingBuffer pingBuffer
}

// NewPlayer returns a new Player.
func NewPlayer(conn net.Conn, server *Server) *Player {
	return &Player{
		Entity: NewEntity("", server),
		conn:   conn,
		state:  stateClosed,
	}
}

// HasPermission implements CommandSender.
func (player *Player) HasPermission(command *Command) bool {
	mask := command.Permissions
	if player.Rank == nil {
		return mask == 0
	}

	if access, ok := player.Rank.Rules[command.Name]; ok {
		return access
	}

	return (mask & player.Rank.Permissions) == mask
}

// HasExtension reports whther the player has the specified CPE extension.
func (player *Player) HasExtension(extension int) bool {
	return player.cpe[extension]
}

// RemoteAddr returns the remote network address as a string.
func (player *Player) RemoteAddr() string {
	addr := player.conn.RemoteAddr()
	host, _, _ := net.SplitHostPort(addr.String())
	return host
}

// Disconnect closes the remote connection.
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

// Kick kicks and disconnects the player.
func (player *Player) Kick(reason string) {
	var packet Packet
	packet.kick(reason)
	player.sendPacket(packet)

	player.Disconnect()
}

// CanReach reports whether the player can reach the block at the specified
// coordinates.
func (player *Player) CanReach(x, y, z int) bool {
	loc := player.location
	dx := math.Min(math.Abs(loc.X-float64(x)), math.Abs(loc.X-float64(x+1)))
	dy := math.Min(math.Abs(loc.Y-float64(y)), math.Abs(loc.Y-float64(y+1)))
	dz := math.Min(math.Abs(loc.Z-float64(z)), math.Abs(loc.Z-float64(z+1)))
	dist := player.level.HackConfig.ReachDistance
	return dx*dx+dy*dy+dz*dz <= dist*dist
}

func (player *Player) HeldBlock() byte {
	return player.heldBlock
}

// SetHeldBlock changes the block that the player is holding.
// lock controls whether the player can change the held block.
func (player *Player) SetHeldBlock(block byte, lock bool) {
	level := player.level
	if player.state == stateGame && player.cpe[CpeHeldBlock] && level != nil {
		var packet Packet
		packet.holdThis(player.convertBlock(block, level), lock)
		player.sendPacket(packet)
	}
}

// SetSelection marks a cuboid selection.
func (player *Player) SetSelection(id byte, label string, box AABB, color color.RGBA) {
	if player.state == stateGame && player.cpe[CpeSelectionCuboid] {
		var packet Packet
		packet.makeSelection(id, label, box, color)
		player.sendPacket(packet)
	}
}

// ResetSelection resets the selection with the specified ID.
func (player *Player) ResetSelection(id byte) {
	if player.state == stateGame && player.cpe[CpeSelectionCuboid] {
		var packet Packet
		packet.removeSelection(id)
		player.sendPacket(packet)
	}
}

func (player *Player) convertColor(code byte) (result byte, ok bool) {
	if (code >= 'a' && code <= 'f') || (code >= '0' && code <= '9') {
		return code, true
	}

	if code >= 'A' && code <= 'F' {
		return code + 32, true
	}

	for _, desc := range player.server.Colors {
		if desc.Code == code {
			if player.cpe[CpeTextColors] {
				return desc.Code, true
			} else {
				return desc.Fallback, true
			}
		}
	}

	return
}

func (player *Player) convertMessage(message string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(message)))
	for i := 0; i < len(message); i++ {
		c := message[i]
		if (c == '%' || c == '&') && i < len(message)-1 {
			i++
			if code, ok := player.convertColor(message[i]); ok {
				buf.Write([]byte{'&', code})
			}
		} else {
			buf.WriteByte(c)
		}
	}

	return buf.String()
}

// SendMessage sends a message to the player.
func (player *Player) SendMessage(message string) {
	player.SendMessageExt(MessageChat, message)
}

// SendMessageExt sends a message with the specified type to the player.
func (player *Player) SendMessageExt(msgType int, message string) {
	if msgType != MessageChat && !player.cpe[CpeMessageTypes] {
		if msgType == MessageAnnouncement {
			msgType = MessageChat
		} else {
			return
		}
	}

	var packet Packet
	message = player.convertMessage(message)
	for _, line := range WordWrap(message, 64) {
		packet.message(msgType, line)
	}

	player.sendPacket(packet)
}

// SetSpawn sets the spawn location of the player to the current player
// location.
func (player *Player) SetSpawn() {
	player.sendSpawn(player.Entity)
}

func (player *Player) sendPacket(packet Packet) {
	if player.state == stateClosed {
		return
	}

	_, err := packet.WriteTo(player.conn)
	if err == io.EOF {
		player.Disconnect()
	}
}

func (player *Player) convertBlock(block byte, level *Level) byte {
	if !player.cpe[CpeBlockDefinitions] {
		if level.BlockDefs != nil {
			if def := level.BlockDefs[block]; def != nil {
				block = def.Fallback
			}
		}

		if block > BlockMaxCPE {
			return BlockAir
		}
	}

	if player.cpeBlockLevel < 1 {
		return FallbackBlock(block)
	}

	return block
}

func (player *Player) sendMOTD(level *Level) {
	op := level.HackConfig.CanPlace[BlockBedrock]

	motd := level.MOTD
	if len(motd) == 0 {
		motd = player.server.Config.MOTD
	}

	var packet Packet
	packet.motd(player, motd, op)
	player.sendPacket(packet)
}

func (player *Player) sendLevel(level *Level) {
	if player.state != stateGame {
		return
	}

	player.sendMOTD(level)

	var conv [BlockMax]byte
	for i := byte(0); i < BlockMax; i++ {
		conv[i] = player.convertBlock(i, level)
	}

	stream := levelStream{player: player}
	stream.reset()
	if player.cpe[CpeFastMap] {
		var packet Packet
		packet.levelInitializeExt(level.Size())
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
		binary.Write(writer, binary.BigEndian, int32(level.Size()))
		for i, block := range level.Blocks {
			stream.percent = byte(i * 100 / len(level.Blocks))
			writer.Write([]byte{conv[block]})
		}
		writer.Close()
	}
	stream.Close()

	player.sendBlockDefinitions(level)
	player.sendInventory(level)
	player.sendEnvConfig(level, EnvPropAll)
	player.sendHackConfig(level)

	var packet Packet
	packet.levelFinalize(level.Width, level.Height, level.Length)
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
	player.resetBlockDefinitions(level)
	player.resetInventory(level)
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

func (player *Player) sendBlockChange(x, y, z int, block byte) {
	level := player.level
	if player.state == stateGame && level != nil {
		var packet Packet
		packet.setBlock(x, y, z, player.convertBlock(block, level))
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

func (player *Player) sendHotkeys() {
	if player.state == stateGame && player.cpe[CpeTextHotKey] {
		var packet Packet
		for _, desc := range player.server.Hotkeys {
			packet.setTextHotKey(&desc)
		}

		player.sendPacket(packet)
	}
}

func (player *Player) sendTextColors() {
	if player.state == stateGame && player.cpe[CpeTextColors] {
		var packet Packet
		for _, desc := range player.server.Colors {
			packet.setTextColor(&desc)
		}

		player.sendPacket(packet)
	}
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

func (player *Player) sendBlockDefinitions(level *Level) {
	if player.state != stateGame || !player.cpe[CpeBlockDefinitions] {
		return
	}

	var packet Packet
	extTex := player.cpe[CpeExtendedTextures]
	for id, def := range level.BlockDefs {
		if def != nil {
			if player.cpe[CpeBlockDefinitionsExt] && def.Shape != 0 {
				packet.defineBlock(byte(id), def, true, extTex)
			} else {
				packet.defineBlock(byte(id), def, false, extTex)
			}
		}
	}

	player.sendPacket(packet)
}

func (player *Player) resetBlockDefinitions(level *Level) {
	if player.state != stateGame || !player.cpe[CpeBlockDefinitions] {
		return
	}

	var packet Packet
	for id, def := range level.BlockDefs {
		if def != nil {
			packet.removeBlockDefinition(byte(id))
		}
	}

	player.sendPacket(packet)
}

func (player *Player) sendInventory(level *Level) {
	if player.state == stateGame && player.cpe[CpeInventoryOrder] {
		var packet Packet
		for id, order := range level.Inventory {
			packet.setInventoryOrder(order, byte(id))
		}

		player.sendPacket(packet)
	}
}

func (player *Player) resetInventory(level *Level) {
	if player.state == stateGame && player.cpe[CpeInventoryOrder] {
		var packet Packet
		for id := range level.Inventory {
			packet.setInventoryOrder(byte(id), byte(id))
		}

		player.sendPacket(packet)
	}
}

func (player *Player) sendEnvConfig(level *Level, mask uint32) {
	if player.state != stateGame {
		return
	}

	var packet Packet
	config := &level.EnvConfig
	if player.cpe[CpeEnvMapAspect] {
		if mask&EnvPropSideBlock != 0 {
			packet.mapEnvProperty(
				0, int32(player.convertBlock(config.SideBlock, level)))
		}
		if mask&EnvPropEdgeBlock != 0 {
			packet.mapEnvProperty(
				1, int32(player.convertBlock(config.EdgeBlock, level)))
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
		if mask&EnvPropTexturePack != 0 {
			packet.mapEnvUrl(config.TexturePack)
		}
	}

	if player.cpe[CpeEnvColors] {
		if mask&EnvPropSkyColor != 0 {
			packet.envSetColor(0, config.SkyColor)
		}
		if mask&EnvPropCloudColor != 0 {
			packet.envSetColor(1, config.CloudColor)
		}
		if mask&EnvPropFogColor != 0 {
			packet.envSetColor(2, config.FogColor)
		}
		if mask&EnvPropAmbientColor != 0 {
			packet.envSetColor(3, config.AmbientColor)
		}
		if mask&EnvPropDiffuseColor != 0 {
			packet.envSetColor(4, config.DiffuseColor)
		}
	}

	if player.cpe[CpeEnvWeatherType] {
		if mask&EnvPropWeather != 0 {
			packet.envWeatherType(config.Weather)
		}
	}

	player.sendPacket(packet)
}

func (player *Player) sendHackConfig(level *Level) {
	if player.state != stateGame {
		return
	}

	var packet Packet
	config := &level.HackConfig
	packet.updateUserType(config.CanPlace[BlockBedrock])
	if player.cpe[CpeClickDistance] {
		packet.clickDistance(config.ReachDistance)
	}
	if player.cpe[CpeHackControl] {
		packet.hackControl(config)
	}
	if player.cpe[CpeBlockPermissions] {
		for i := 0; i < BlockCount; i++ {
			packet.setBlockPermission(byte(i), config.CanPlace[i], config.CanBreak[i])
		}
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
		var size int
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

	if player.server.FindEntity(player.name) != nil {
		player.Kick("Already logged in!")
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

	if player.cpe[CpeBlockDefinitions] {
		player.maxBlockID = BlockMax
	} else if player.cpe[CpeCustomBlocks] && player.cpeBlockLevel == 1 {
		player.maxBlockID = BlockMaxCPE
	} else {
		player.maxBlockID = BlockMaxClassic
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

	player.sendHotkeys()
	player.sendTextColors()
	player.server.BroadcastMessage(ColorYellow + player.name + " has joined the game!")

	if player.server.MainLevel != nil {
		player.TeleportLevel(player.server.MainLevel)
	}

	player.pingTicker = time.NewTicker(2 * time.Second)
	go func() {
		for range player.pingTicker.C {
			var packet Packet
			if player.cpe[CpeTwoWayPing] {
				packet.twoWayPing(1, player.pingBuffer.Next())
			} else {
				packet.ping()
			}

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

	key := trimString(packet.VerificationKey)
	if player.server.Config.Verify {
		if !player.verify([]byte(key)) {
			player.Kick("Login failed!")
			return
		}
	}

	if packet.Type == 0x42 {
		player.sendCPE()
	} else {
		player.login()
	}
}

func (player *Player) revertBlock(x, y, z int) {
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
	x, y, z := int(packet.X), int(packet.Y), int(packet.Z)
	block := packet.BlockType

	level := player.level
	if !level.InBounds(x, y, z) {
		return
	}

	if !player.CanReach(x, y, z) {
		player.SendMessage("You cannot build that far away.")
		player.revertBlock(x, y, z)
		return
	}

	switch packet.Mode {
	case 0x00:
		oldBlock := level.GetBlock(x, y, z)
		if !level.HackConfig.CanBreak[oldBlock] {
			player.SendMessage("You cannot break that block.")
			player.revertBlock(x, y, z)
			return
		}

		event := &EventBlockBreak{player, level, oldBlock, x, y, z, false}
		player.server.FireEvent(EventTypeBlockBreak, &event)
		if event.Cancel {
			player.revertBlock(x, y, z)
			return
		}

		level.SetBlock(x, y, z, BlockAir)

	case 0x01:
		if block > player.maxBlockID {
			player.SendMessage("Invalid block!")
			player.revertBlock(x, y, z)
			return
		}

		if !level.HackConfig.CanPlace[block] {
			player.SendMessage("You cannot place that block.")
			player.revertBlock(x, y, z)
			return
		}

		event := &EventBlockPlace{player, level, block, x, y, z, false}
		player.server.FireEvent(EventTypeBlockPlace, &event)
		if event.Cancel {
			player.revertBlock(x, y, z)
			return
		}

		level.SetBlock(x, y, z, block)
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
		player.heldBlock = packet0.PlayerID
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

		event := EventPlayerChat{player, players, message, "%s%s: &f%s", false}
		player.server.FireEvent(EventTypePlayerChat, &event)
		if event.Cancel {
			return
		}

		var tag string
		if player.Rank != nil {
			tag = player.Rank.Tag
		}

		message = fmt.Sprintf(event.Format, tag, player.Nickname, event.Message)

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

	player.remExtensions = int(packet.ExtensionCount)
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
		int(packet.BlockX), int(packet.BlockY), int(packet.BlockZ),
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

	case 1:
		player.pingBuffer.Update(packet.Data)
	}
}
