// Copyright 2017-2018 Andrew Goulas
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
	PacketTypeIdentification            = 0x00
	PacketTypePing                      = 0x01
	PacketTypeLevelInitialize           = 0x02
	PacketTypeLevelDataChunk            = 0x03
	PacketTypeLevelFinalize             = 0x04
	PacketTypeSetBlockClient            = 0x05
	PacketTypeSetBlock                  = 0x06
	PacketTypeSpawnPlayer               = 0x07
	PacketTypePlayerTeleport            = 0x08
	PacketTypePositionOrientationUpdate = 0x09
	PacketTypePositionUpdate            = 0x0a
	PacketTypeOrientationUpdate         = 0x0b
	PacketTypeDespawnPlayer             = 0x0c
	PacketTypeMessage                   = 0x0d
	PacketTypeDisconnect                = 0x0e
	PacketTypeUpdateUserType            = 0x0f

	PacketTypeExtInfo                 = 0x10
	PacketTypeExtEntry                = 0x11
	PacketTypeSetClickDistance        = 0x12
	PacketTypeCustomBlockSupportLevel = 0x13
	PacketTypeExtAddPlayerName        = 0x16
	PacketTypeExtRemovePlayerName     = 0x18
	PacketTypeChangeModel             = 0x1d
	PacketTypeEnvSetMapAppearance2    = 0x1e
	PacketTypeEnvSetWeatherType       = 0x1f
	PacketTypeExtAddEntity2           = 0x21
)

type PacketClientIdentification struct {
	PacketID        byte
	ProtocolVersion byte
	Name            [64]byte
	VerificationKey [64]byte
	Type            byte
}

type PacketServerIdentification struct {
	PacketID        byte
	ProtocolVersion byte
	Name            [64]byte
	MOTD            [64]byte
	UserType        byte
}

type PacketPing struct {
	PacketID byte
}

type PacketLevelInitialize struct {
	PacketID byte
}

type PacketLevelDataChunk struct {
	PacketID        byte
	ChunkLength     int16
	ChunkData       [1024]byte
	PercentComplete byte
}

type PacketLevelFinalize struct {
	PacketID byte
	X, Y, Z  int16
}

type PacketSetBlockClient struct {
	PacketID  byte
	X, Y, Z   int16
	Mode      byte
	BlockType byte
}

type PacketSetBlock struct {
	PacketID  byte
	X, Y, Z   int16
	BlockType byte
}

type PacketSpawnPlayer struct {
	PacketID   byte
	PlayerID   byte
	Name       [64]byte
	X, Y, Z    int16
	Yaw, Pitch byte
}

type PacketPlayerTeleport struct {
	PacketID   byte
	PlayerID   byte
	X, Y, Z    int16
	Yaw, Pitch byte
}

type PacketPositionOrientationUpdate struct {
	PacketID   byte
	PlayerID   byte
	X, Y, Z    byte
	Yaw, Pitch byte
}

type PacketPositionUpdate struct {
	PacketID byte
	PlayerID byte
	X, Y, Z  byte
}

type PacketOrientationUpdate struct {
	PacketID   byte
	PlayerID   byte
	Yaw, Pitch byte
}

type PacketDespawnPlayer struct {
	PacketID byte
	PlayerID byte
}

type PacketMessage struct {
	PacketID byte
	PlayerID byte
	Message  [64]byte
}

type PacketDisconnect struct {
	PacketID byte
	Reason   [64]byte
}

type PacketUpdateUserType struct {
	PacketID byte
	UserType byte
}

type PacketExtInfo struct {
	PacketID       byte
	AppName        [64]byte
	ExtensionCount int16
}

type PacketExtEntry struct {
	PacketID byte
	ExtName  [64]byte
	Version  int32
}

type PacketSetClickDistance struct {
	PacketID byte
	Distance int16
}

type PacketCustomBlockSupportLevel struct {
	PacketID     byte
	SupportLevel byte
}

type PacketExtAddPlayerName struct {
	PacketID   byte
	NameID     int16
	PlayerName [64]byte
	ListName   [64]byte
	GroupName  [64]byte
	GroupRank  byte
}

type PacketExtRemovePlayerName struct {
	PacketID byte
	NameID   int16
}

type PacketChangeModel struct {
	PacketID  byte
	EntityID  byte
	ModelName [64]byte
}

type PacketEnvSetMapAppearance2 struct {
	PacketID              byte
	TexturePackURL        [64]byte
	SideBlock, EdgeBlock  byte
	SideLevel, CloudLevel int16
	MaxViewDistance       int16
}

type PacketEnvSetWeatherType struct {
	PacketID    byte
	WeatherType byte
}

type PacketExtAddEntity2 struct {
	PacketID    byte
	EntityID    byte
	DisplayName [64]byte
	SkinName    [64]byte
	X, Y, Z     int16
	Yaw, Pitch  byte
}

func PadString(str string) [64]byte {
	var result [64]byte
	copy(result[:], str)
	if len(str) < 64 {
		copy(result[len(str):], bytes.Repeat([]byte{' '}, 64-len(str)))
	}

	return result
}

func TrimString(str [64]byte) string {
	return strings.TrimRight(string(str[:]), " ")
}
