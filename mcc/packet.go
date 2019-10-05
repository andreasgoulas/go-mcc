// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package mcc

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"math"
	"strings"
	"time"
)

const (
	CpeClickDistance = iota
	CpeCustomBlocks
	CpeHeldBlock
	CpeTextHotKey
	CpeExtPlayerList
	CpeEnvColors
	CpeSelectionCuboid
	CpeBlockPermissions
	CpeChangeModel
	CpeEnvWeatherType
	CpeHackControl
	CpeMessageTypes
	CpePlayerClick
	CpeLongerMessages
	CpeBlockDefinitions
	CpeBlockDefinitionsExt
	CpeBulkBlockUpdate
	CpeTextColors
	CpeEnvMapAspect
	CpeEntityProperty
	CpeExtEntityPositions
	CpeTwoWayPing
	CpeInventoryOrder
	CpeInstantMOTD
	CpeFastMap
	CpeExtendedTextures

	CpeMax   = CpeExtendedTextures
	CpeCount = CpeMax + 1
)

type ExtEntry struct {
	Name    string
	Version int
}

var Extensions = [CpeCount]ExtEntry{
	{"ClickDistance", 1},
	{"CustomBlocks", 1},
	{"HeldBlock", 1},
	{"TextHotKey", 1},
	{"ExtPlayerList", 2},
	{"EnvColors", 1},
	{"SelectionCuboid", 1},
	{"BlockPermissions", 1},
	{"ChangeModel", 1},
	{"EnvWeatherType", 1},
	{"HackControl", 1},
	{"MessageTypes", 1},
	{"PlayerClick", 1},
	{"LongerMessages", 1},
	{"BlockDefinitions", 1},
	{"BlockDefinitionsExt", 2},
	{"BulkBlockUpdate", 1},
	{"TextColors", 1},
	{"EnvMapAspect", 1},
	{"EntityProperty", 1},
	{"ExtEntityPositions", 1},
	{"TwoWayPing", 1},
	{"InventoryOrder", 1},
	{"InstantMOTD", 1},
	{"FastMap", 1},
	{"ExtendedTextures", 1},
}

const (
	packetTypeIdentification            = 0x00
	packetTypePing                      = 0x01
	packetTypeLevelInitialize           = 0x02
	packetTypeLevelDataChunk            = 0x03
	packetTypeLevelFinalize             = 0x04
	packetTypeSetBlockClient            = 0x05
	packetTypeSetBlock                  = 0x06
	packetTypeAddEntity                 = 0x07
	packetTypePlayerTeleport            = 0x08
	packetTypePositionOrientationUpdate = 0x09
	packetTypePositionUpdate            = 0x0a
	packetTypeOrientationUpdate         = 0x0b
	packetTypeRemoveEntity              = 0x0c
	packetTypeMessage                   = 0x0d
	packetTypeKick                      = 0x0e
	packetTypeUpdateUserType            = 0x0f
	packetTypeExtInfo                   = 0x10
	packetTypeExtEntry                  = 0x11
	packetTypeSetClickDistance          = 0x12
	packetTypeCustomBlockSupportLevel   = 0x13
	packetTypeHoldThis                  = 0x14
	packetTypeSetTextHotKey             = 0x15
	packetTypeExtAddPlayerName          = 0x16
	packetTypeExtRemovePlayerName       = 0x18
	packetTypeEnvSetColor               = 0x19
	packetTypeMakeSelection             = 0x1a
	packetTypeRemoveSelection           = 0x1b
	packetTypeSetBlockPermission        = 0x1c
	packetTypeChangeModel               = 0x1d
	packetTypeEnvSetWeatherType         = 0x1f
	packetTypeHackControl               = 0x20
	packetTypeExtAddEntity2             = 0x21
	packetTypePlayerClicked             = 0x22
	packetTypeDefineBlock               = 0x23
	packetTypeRemoveBlockDefinition     = 0x24
	packetTypeDefineBlockExt            = 0x25
	packetTypeBulkBlockUpdate           = 0x26
	packetTypeSetTextColor              = 0x27
	packetTypeSetMapEnvUrl              = 0x28
	packetTypeSetMapEnvProperty         = 0x29
	packetTypeSetEntityProperty         = 0x2a
	packetTypeTwoWayPing                = 0x2b
	packetTypeSetInventoryOrder         = 0x2c
)

func padString(str string) [64]byte {
	var result [64]byte
	copy(result[:], str)
	if len(str) < 64 {
		copy(result[len(str):], bytes.Repeat([]byte{' '}, 64-len(str)))
	}

	return result
}

func trimString(str [64]byte) string {
	return strings.TrimRight(string(str[:]), " ")
}

type packet struct {
	bytes.Buffer
}

func (packet *packet) position(location Location, extPos bool) {
	if extPos {
		binary.Write(packet, binary.BigEndian, struct{ X, Y, Z int32 }{
			int32(location.X * 32),
			int32(location.Y * 32),
			int32(location.Z * 32),
		})
	} else {
		binary.Write(packet, binary.BigEndian, struct{ X, Y, Z int16 }{
			int16(location.X * 32),
			int16(location.Y * 32),
			int16(location.Z * 32),
		})
	}
}

func (packet *packet) textureID(textureID int, extTex bool) {
	if extTex {
		binary.Write(packet, binary.BigEndian, int16(textureID))
	} else {
		packet.WriteByte(byte(textureID))
	}
}

func (packet *packet) motd(player *Player, motd string, op bool) {
	userType := byte(0x00)
	if op {
		userType = 0x64
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID        byte
		ProtocolVersion byte
		Name            [64]byte
		MOTD            [64]byte
		UserType        byte
	}{
		packetTypeIdentification,
		0x07,
		padString(player.server.Config.Name),
		padString(motd),
		userType,
	})
}

func (packet *packet) ping() {
	packet.WriteByte(packetTypePing)
}

func (packet *packet) levelInitialize() {
	packet.WriteByte(packetTypeLevelInitialize)
}

func (packet *packet) levelInitializeExt(size int) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		Size     int32
	}{packetTypeLevelInitialize, int32(size)})
}

func (packet *packet) levelFinalize(x, y, z int) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		X, Y, Z  int16
	}{packetTypeLevelFinalize, int16(x), int16(y), int16(z)})
}

func (packet *packet) setBlock(x, y, z int, block byte) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID  byte
		X, Y, Z   int16
		BlockType byte
	}{packetTypeSetBlock, int16(x), int16(y), int16(z), block})
}

func (packet *packet) addEntity(entity *Entity, self bool, extPos bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	location := entity.location
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		PlayerID byte
		Name     [64]byte
	}{packetTypeAddEntity, id, padString(entity.DisplayName)})

	packet.position(location, extPos)
	binary.Write(packet, binary.BigEndian, struct{ Yaw, Pitch byte }{
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *packet) teleport(entity *Entity, self bool, extPos bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	location := entity.location
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		PlayerID byte
	}{packetTypePlayerTeleport, id})

	packet.position(location, extPos)
	binary.Write(packet, binary.BigEndian, struct{ Yaw, Pitch byte }{
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *packet) positionOrientationUpdate(entity *Entity) {
	location := entity.location
	lastLocation := entity.lastLocation
	binary.Write(packet, binary.BigEndian, struct {
		PacketID   byte
		PlayerID   byte
		X, Y, Z    byte
		Yaw, Pitch byte
	}{
		packetTypePositionOrientationUpdate,
		entity.id,
		byte((location.X - lastLocation.X) * 32),
		byte((location.Y - lastLocation.Y) * 32),
		byte((location.Z - lastLocation.Z) * 32),
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *packet) positionUpdate(entity *Entity) {
	location := entity.location
	lastLocation := entity.lastLocation
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		PlayerID byte
		X, Y, Z  byte
	}{
		packetTypePositionUpdate,
		entity.id,
		byte((location.X - lastLocation.X) * 32),
		byte((location.Y - lastLocation.Y) * 32),
		byte((location.Z - lastLocation.Z) * 32),
	})
}

func (packet *packet) orientationUpdate(entity *Entity) {
	location := entity.location
	binary.Write(packet, binary.BigEndian, struct {
		PacketID   byte
		PlayerID   byte
		Yaw, Pitch byte
	}{
		packetTypeOrientationUpdate,
		entity.id,
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *packet) removeEntity(entity *Entity, self bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		PlayerID byte
	}{packetTypeRemoveEntity, id})
}

func (packet *packet) message(msgType int, message string) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		PlayerID byte
		Message  [64]byte
	}{packetTypeMessage, byte(msgType), padString(message)})
}

func (packet *packet) kick(reason string) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		Reason   [64]byte
	}{packetTypeKick, padString(reason)})
}

func (packet *packet) updateUserType(op bool) {
	userType := byte(0x00)
	if op {
		userType = 0x64
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		UserType byte
	}{packetTypeUpdateUserType, userType})
}

func (packet *packet) extInfo() {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID       byte
		AppName        [64]byte
		ExtensionCount int16
	}{packetTypeExtInfo, padString(ServerSoftware), int16(len(Extensions))})
}

func (packet *packet) extEntry(entry *ExtEntry) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		ExtName  [64]byte
		Version  int32
	}{packetTypeExtEntry, padString(entry.Name), int32(entry.Version)})
}

func (packet *packet) clickDistance(dist float64) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		Distance int16
	}{packetTypeSetClickDistance, int16(dist * 32)})
}

func (packet *packet) customBlockSupportLevel(level byte) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID     byte
		SupportLevel byte
	}{packetTypeCustomBlockSupportLevel, level})
}

func (packet *packet) holdThis(block byte, lock bool) {
	preventChange := byte(0)
	if lock {
		preventChange = 1
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID      byte
		BlockToHold   byte
		PreventChange byte
	}{packetTypeHoldThis, block, preventChange})
}

func (packet *packet) setTextHotKey(hotkey *HotkeyDesc) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		Label    [64]byte
		Action   [64]byte
		KeyCode  int32
		KeyMods  byte
	}{
		packetTypeSetTextHotKey,
		padString(hotkey.Label),
		padString(hotkey.Action),
		int32(hotkey.Key),
		hotkey.KeyMods,
	})
}

func (packet *packet) extAddPlayerName(entity *Entity, self bool) {
	id := int16(entity.id)
	if self {
		id = 0xff
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID   byte
		NameID     int16
		PlayerName [64]byte
		ListName   [64]byte
		GroupName  [64]byte
		GroupRank  byte
	}{
		packetTypeExtAddPlayerName,
		id,
		padString(entity.name),
		padString(entity.ListName),
		padString(entity.GroupName),
		entity.GroupRank,
	})
}

func (packet *packet) extRemovePlayerName(entity *Entity, self bool) {
	id := int16(entity.id)
	if self {
		id = 0xff
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		NameID   int16
	}{packetTypeExtRemovePlayerName, id})
}

func (packet *packet) makeSelection(id byte, label string, box AABB, color color.RGBA) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID               byte
		SelectionID            byte
		Label                  [64]byte
		StartX, StartY, StartZ int16
		EndX, EndY, Endz       int16
		R, G, B, Opacity       int16
	}{
		packetTypeMakeSelection,
		id,
		padString(label),
		int16(box.Min.X), int16(box.Min.Y), int16(box.Min.Z),
		int16(box.Max.X), int16(box.Max.Y), int16(box.Max.Z),
		int16(color.R), int16(color.G), int16(color.B), int16(color.A),
	})
}

func (packet *packet) removeSelection(id byte) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID    byte
		SelectionID byte
	}{packetTypeRemoveSelection, id})
}

func (packet *packet) envSetColor(id byte, color color.RGBA) {
	data := struct {
		packetId byte
		Variable byte
		R, G, B  int16
	}{packetTypeEnvSetColor, id, -1, -1, -1}
	if color.A != 0 {
		data.R = int16(color.R)
		data.G = int16(color.G)
		data.B = int16(color.B)
	}

	binary.Write(packet, binary.BigEndian, &data)
}

func (packet *packet) setBlockPermission(id byte, canPlace, canBreak bool) {
	data := struct {
		PacketID       byte
		BlockType      byte
		AllowPlacement byte
		AllowDeletion  byte
	}{packetTypeSetBlockPermission, id, 0, 0}
	if canPlace {
		data.AllowPlacement = 1
	}
	if canBreak {
		data.AllowDeletion = 1
	}

	binary.Write(packet, binary.BigEndian, data)
}

func (packet *packet) changeModel(entity *Entity, self bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID  byte
		EntityID  byte
		ModelName [64]byte
	}{packetTypeChangeModel, id, padString(entity.Model)})
}

func (packet *packet) envWeatherType(weather byte) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID    byte
		WeatherType byte
	}{packetTypeEnvSetWeatherType, weather})
}

func (packet *packet) hackControl(config *HackConfig) {
	data := struct {
		PacketID        byte
		Flying          byte
		NoClip          byte
		Speeding        byte
		SpawnControl    byte
		ThirdPersonView byte
		JumpHeight      int16
	}{packetTypeHackControl, 0, 0, 0, 0, 0, -1}

	if config.Flying {
		data.Flying = 1
	}
	if config.NoClip {
		data.NoClip = 1
	}
	if config.Speeding {
		data.Speeding = 1
	}
	if config.SpawnControl {
		data.SpawnControl = 1
	}
	if config.ThirdPersonView {
		data.ThirdPersonView = 1
	}
	if config.JumpHeight >= 0 {
		data.JumpHeight = int16(config.JumpHeight * 32)
	}

	binary.Write(packet, binary.BigEndian, &data)
}

func (packet *packet) extAddEntity2(entity *Entity, self bool, extPos bool) {
	id := entity.id
	if self {
		id = 0xff
	}

	location := entity.location
	binary.Write(packet, binary.BigEndian, struct {
		PacketID    byte
		EntityID    byte
		DisplayName [64]byte
		SkinName    [64]byte
	}{
		packetTypeExtAddEntity2,
		id,
		padString(entity.DisplayName),
		padString(entity.SkinName),
	})

	packet.position(location, extPos)
	binary.Write(packet, binary.BigEndian, struct{ Yaw, Pitch byte }{
		byte(location.Yaw * 256 / 360),
		byte(location.Pitch * 256 / 360),
	})
}

func (packet *packet) defineBlock(id byte, block *BlockDefinition, ext bool, extTex bool) {
	packetID := byte(packetTypeDefineBlock)
	if ext {
		packetID = packetTypeDefineBlockExt
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID      byte
		BlockID       byte
		Name          [64]byte
		Solidity      byte
		MovementSpeed byte
	}{
		packetID,
		id,
		padString(block.Name),
		block.CollideMode,
		byte(64*math.Log2(block.Speed) + 128),
	})

	packet.textureID(block.Textures[FacePosY], extTex)
	if ext {
		packet.textureID(block.Textures[FaceNegX], extTex)
		packet.textureID(block.Textures[FacePosX], extTex)
		packet.textureID(block.Textures[FaceNegZ], extTex)
		packet.textureID(block.Textures[FacePosZ], extTex)
	} else {
		packet.textureID(block.Textures[FacePosX], extTex)
	}
	packet.textureID(block.Textures[FaceNegY], extTex)

	transmitsLight := byte(1)
	if block.BlockLight {
		transmitsLight = 0
	}

	fullBright := byte(0)
	if block.FullBright {
		fullBright = 1
	}

	binary.Write(packet, binary.BigEndian, struct {
		TransmitsLight byte
		WalkSound      byte
		FullBright     byte
	}{transmitsLight, block.WalkSound, fullBright})

	if ext {
		aabb := block.AABB
		binary.Write(packet, binary.BigEndian, struct {
			MinX, MinY, MinZ byte
			MaxX, MaxY, MaxZ byte
		}{
			byte(aabb.Min.X), byte(aabb.Min.Y), byte(aabb.Min.Z),
			byte(aabb.Max.X), byte(aabb.Max.Y), byte(aabb.Max.Z),
		})
	} else {
		packet.WriteByte(block.Shape)
	}

	binary.Write(packet, binary.BigEndian, struct {
		BlockDraw        byte
		FogDensity       byte
		FogR, FogG, FogB byte
	}{
		block.DrawMode,
		block.FogDensity,
		block.Fog.R, block.Fog.G, block.Fog.B,
	})
}

func (packet *packet) removeBlockDefinition(id byte) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		BlockID  byte
	}{packetTypeRemoveBlockDefinition, id})
}

func (packet *packet) bulkBlockUpdate(indices []int32, blocks []byte) {
	data := struct {
		PacketID byte
		Count    byte
		Indices  [256]int32
		Blocks   [256]byte
	}{
		packetTypeBulkBlockUpdate,
		byte(len(indices)),
		[256]int32{},
		[256]byte{},
	}

	copy(data.Indices[:], indices)
	copy(data.Blocks[:], blocks)
	binary.Write(packet, binary.BigEndian, data)
}

func (packet *packet) setTextColor(color *ColorDesc) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID   byte
		R, G, B, A byte
		Code       byte
	}{
		packetTypeSetTextColor,
		color.R, color.G, color.B, color.A,
		color.Code,
	})
}

func (packet *packet) mapEnvUrl(texturePack string) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID       byte
		TexturePackURL [64]byte
	}{packetTypeSetMapEnvUrl, padString(texturePack)})
}

func (packet *packet) mapEnvProperty(id byte, value int32) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		Type     byte
		Value    int32
	}{packetTypeSetMapEnvProperty, id, value})
}

func (packet *packet) entityProperty(entity *Entity, self bool, prop byte, value int32) {
	id := entity.id
	if self {
		id = 0xff
	}

	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		EntityID byte
		Type     byte
		Value    int32
	}{packetTypeSetEntityProperty, id, prop, value})
}

func (packet *packet) twoWayPing(dir byte, data int16) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID  byte
		Direction byte
		Data      int16
	}{packetTypeTwoWayPing, dir, data})
}

func (packet *packet) setInventoryOrder(order byte, block byte) {
	binary.Write(packet, binary.BigEndian, struct {
		PacketID byte
		Order    byte
		BlockID  byte
	}{packetTypeSetInventoryOrder, order, block})
}

type levelStream struct {
	player  *Player
	packet  packet
	index   int
	percent byte
}

func (stream *levelStream) reset() {
	stream.packet.Reset()
	stream.packet.Write([]byte{packetTypeLevelDataChunk, 0, 0})
	stream.index = 0
}

func (stream *levelStream) send() {
	if stream.index < 1024 {
		stream.packet.Write(make([]byte, 1024-stream.index))
	}
	stream.packet.Write([]byte{stream.percent})

	buf := stream.packet.Bytes()
	binary.BigEndian.PutUint16(buf[1:], uint16(stream.index))

	stream.player.sendPacket(stream.packet)
	stream.reset()
}

func (stream *levelStream) Close() {
	if stream.index > 0 {
		stream.send()
	}
}

func (stream *levelStream) Write(p []byte) (int, error) {
	offset := 0
	count := len(p)
	for count > 0 {
		size := min(1024-stream.index, count)
		stream.packet.Write(p[offset : offset+size])

		stream.index += size
		offset += size
		count -= size

		if stream.index == 1024 {
			stream.send()
		}
	}

	return len(p), nil
}

type pingEntry struct {
	data           int16
	sent, received time.Time
}

type pingBuffer struct {
	entries [10]pingEntry
	index   int
}

func (buf *pingBuffer) Next() int16 {
	data := buf.entries[buf.index].data + 1
	buf.index = (buf.index + 1) % len(buf.entries)
	buf.entries[buf.index] = pingEntry{
		data: data,
		sent: time.Now(),
	}

	return data
}

func (buf *pingBuffer) Update(data int16) {
	for i, entry := range buf.entries {
		if entry.data == data {
			buf.entries[i].received = time.Now()
			break
		}
	}
}

func (buf *pingBuffer) Average() (d time.Duration) {
	count := int64(0)
	var sum time.Duration
	for _, entry := range buf.entries {
		if entry.received.After(entry.sent) {
			sum += entry.received.Sub(entry.sent)
			count++
		}
	}

	if count > 0 {
		return time.Duration(int64(sum) / (2 * count))
	}

	return
}
