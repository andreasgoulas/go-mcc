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
	"unicode"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
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
	packetTypeExtAddPlayerName        = 0x16
	packetTypeExtRemovePlayerName     = 0x18
	packetTypeChangeModel             = 0x1d
	packetTypeEnvSetMapAppearance2    = 0x1e
	packetTypeEnvSetWeatherType       = 0x1f
	packetTypeExtAddEntity2           = 0x21
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

type packetChangeModel struct {
	PacketID  byte
	EntityID  byte
	ModelName [64]byte
}

type packetEnvSetMapAppearance2 struct {
	PacketID              byte
	TexturePackURL        [64]byte
	SideBlock, EdgeBlock  byte
	SideLevel, CloudLevel int16
	MaxViewDistance       int16
}

type packetEnvSetWeatherType struct {
	PacketID    byte
	WeatherType byte
}

type packetExtAddEntity2 struct {
	PacketID    byte
	EntityID    byte
	DisplayName [64]byte
	skinName    [64]byte
	X, Y, Z     int16
	Yaw, Pitch  byte
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
