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
	"strings"
)

const (
	CpeClickDistance = iota
	CpeCustomBlocks
	CpeHeldBlock
	CpeExtPlayerList
	CpeLongerMessages
	CpeSelectionCuboid
	CpeChangeModel
	CpeEnvWeatherType
	CpeHackControl
	CpeMessageTypes
	CpePlayerClick
	CpeBulkBlockUpdate
	CpeEnvMapAspect
	CpeTwoWayPing
	CpeInstantMOTD
	CpeFastMap

	CpeMax   = CpeFastMap
	CpeCount = CpeMax + 1
)

var Extensions = [CpeCount]struct {
	Name    string
	Version int
}{
	{"ClickDistance", 1},
	{"CustomBlocks", 1},
	{"HeldBlock", 1},
	{"ExtPlayerList", 2},
	{"LongerMessages", 1},
	{"SelectionCuboid", 1},
	{"ChangeModel", 1},
	{"EnvWeatherType", 1},
	{"HackControl", 1},
	{"MessageTypes", 1},
	{"PlayerClick", 1},
	{"BulkBlockUpdate", 1},
	{"EnvMapAspect", 1},
	{"TwoWayPing", 1},
	{"InstantMOTD", 1},
	{"FastMap", 1},
}

const (
	packetTypeIdentification            = 0x00
	packetTypePing                      = 0x01
	packetTypeLevelInitialize           = 0x02
	packetTypeLevelDataChunk            = 0x03
	packetTypeLevelFinalize             = 0x04
	packetTypeSetBlockClient            = 0x05
	packetTypeSetBlock                  = 0x06
	packetTypeSpawnPlayer               = 0x07
	packetTypePlayerTeleport            = 0x08
	packetTypePositionOrientationUpdate = 0x09
	packetTypePositionUpdate            = 0x0a
	packetTypeOrientationUpdate         = 0x0b
	packetTypeDespawnPlayer             = 0x0c
	packetTypeMessage                   = 0x0d
	packetTypeDisconnect                = 0x0e
	packetTypeUpdateUserType            = 0x0f

	packetTypeExtInfo                 = 0x10
	packetTypeExtEntry                = 0x11
	packetTypeSetClickDistance        = 0x12
	packetTypeCustomBlockSupportLevel = 0x13
	packetTypeHoldThis                = 0x14
	packetTypeExtAddPlayerName        = 0x16
	packetTypeExtRemovePlayerName     = 0x18
	packetTypeMakeSelection           = 0x1a
	packetTypeRemoveSelection         = 0x1b
	packetTypeChangeModel             = 0x1d
	packetTypeEnvSetWeatherType       = 0x1f
	packetTypeHackControl             = 0x20
	packetTypeExtAddEntity2           = 0x21
	packetTypePlayerClicked           = 0x22
	packetTypeBulkBlockUpdate         = 0x26
	packetTypeSetMapEnvUrl            = 0x28
	packetTypeSetMapEnvProperty       = 0x29
	packetTypeTwoWayPing              = 0x2b
)

type packetClientIdentification struct {
	PacketID        byte
	ProtocolVersion byte
	Name            [64]byte
	VerificationKey [64]byte
	Type            byte
}

type packetServerIdentification struct {
	PacketID        byte
	ProtocolVersion byte
	Name            [64]byte
	MOTD            [64]byte
	UserType        byte
}

type packetPing struct {
	PacketID byte
}

type packetLevelInitialize struct {
	PacketID byte
}

type packetLevelInitializeExt struct {
	PacketID byte
	Size     int32
}

type packetLevelDataChunk struct {
	PacketID        byte
	ChunkLength     int16
	ChunkData       [1024]byte
	PercentComplete byte
}

type packetLevelFinalize struct {
	PacketID byte
	X, Y, Z  int16
}

type packetSetBlockClient struct {
	PacketID  byte
	X, Y, Z   int16
	Mode      byte
	BlockType byte
}

type packetSetBlock struct {
	PacketID  byte
	X, Y, Z   int16
	BlockType byte
}

type packetSpawnPlayer struct {
	PacketID   byte
	PlayerID   byte
	Name       [64]byte
	X, Y, Z    int16
	Yaw, Pitch byte
}

type packetPlayerTeleport struct {
	PacketID   byte
	PlayerID   byte
	X, Y, Z    int16
	Yaw, Pitch byte
}

type packetPositionOrientationUpdate struct {
	PacketID   byte
	PlayerID   byte
	X, Y, Z    byte
	Yaw, Pitch byte
}

type packetPositionUpdate struct {
	PacketID byte
	PlayerID byte
	X, Y, Z  byte
}

type packetOrientationUpdate struct {
	PacketID   byte
	PlayerID   byte
	Yaw, Pitch byte
}

type packetDespawnPlayer struct {
	PacketID byte
	PlayerID byte
}

type packetMessage struct {
	PacketID byte
	PlayerID byte
	Message  [64]byte
}

type packetDisconnect struct {
	PacketID byte
	Reason   [64]byte
}

type packetUpdateUserType struct {
	PacketID byte
	UserType byte
}

type packetExtInfo struct {
	PacketID       byte
	AppName        [64]byte
	ExtensionCount int16
}

type packetExtEntry struct {
	PacketID byte
	ExtName  [64]byte
	Version  int32
}

type packetSetClickDistance struct {
	PacketID byte
	Distance int16
}

type packetCustomBlockSupportLevel struct {
	PacketID     byte
	SupportLevel byte
}

type packetHoldThis struct {
	PacketID      byte
	BlockToHold   byte
	PreventChange byte
}

type packetExtAddPlayerName struct {
	PacketID   byte
	NameID     int16
	PlayerName [64]byte
	ListName   [64]byte
	GroupName  [64]byte
	GroupRank  byte
}

type packetExtRemovePlayerName struct {
	PacketID byte
	NameID   int16
}

type packetMakeSelection struct {
	PacketID               byte
	SelectionID            byte
	Label                  [64]byte
	StartX, StartY, StartZ int16
	EndX, EndY, Endz       int16
	R, G, B, Opacity       int16
}

type packetRemoveSelection struct {
	PacketID    byte
	SelectionID byte
}

type packetChangeModel struct {
	PacketID  byte
	EntityID  byte
	ModelName [64]byte
}

type packetEnvSetWeatherType struct {
	PacketID    byte
	WeatherType byte
}

type packetHackControl struct {
	PacketID        byte
	Flying          byte
	NoClip          byte
	Speeding        byte
	SpawnControl    byte
	ThirdPersonView byte
	JumpHeight      int16
}

type packetExtAddEntity2 struct {
	PacketID    byte
	EntityID    byte
	DisplayName [64]byte
	skinName    [64]byte
	X, Y, Z     int16
	Yaw, Pitch  byte
}

type packetPlayerClicked struct {
	PacketID               byte
	Button, Action         byte
	Yaw, Pitch             int16
	TargetID               byte
	BlockX, BlockY, BlockZ int16
	BlockFace              byte
}

type packetBulkBlockUpdate struct {
	PacketID byte
	Count    byte
	Indices  [256]int32
	Blocks   [256]byte
}

type packetSetMapEnvUrl struct {
	PacketID       byte
	TexturePackURL [64]byte
}

type packetSetMapEnvProperty struct {
	PacketID byte
	Type     byte
	Value    int32
}

type packetTwoWayPing struct {
	PacketID  byte
	Direction byte
	Data      int16
}

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
