// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package mcc

import (
	"image/color"
)

const (
	BlockAir         = 0
	BlockStone       = 1
	BlockGrass       = 2
	BlockDirt        = 3
	BlockCobblestone = 4
	BlockWood        = 5
	BlockSapling     = 6
	BlockBedrock     = 7
	BlockActiveWater = 8
	BlockWater       = 9
	BlockActiveLava  = 10
	BlockLava        = 11
	BlockSand        = 12
	BlockGravel      = 13
	BlockGoldOre     = 14
	BlockIronOre     = 15
	BlockCoal        = 16
	BlockLog         = 17
	BlockLeaves      = 18
	BlockSponge      = 19
	BlockGlass       = 20
	BlockRed         = 21
	BlockOrange      = 22
	BlockYellow      = 23
	BlockLime        = 24
	BlockGreen       = 25
	BlockAqua        = 26
	BlockCyan        = 27
	BlockBlue        = 28
	BlockPurple      = 29
	BlockIndigo      = 30
	BlockViolet      = 31
	BlockMagenta     = 32
	BlockPink        = 33
	BlockBlack       = 34
	BlockGray        = 35
	BlockWhite       = 36
	BlockDandelion   = 37
	BlockRose        = 38
	BlockBrownShroom = 39
	BlockRedShroom   = 40
	BlockGold        = 41
	BlockIron        = 42
	BlockDoubleSlab  = 43
	BlockSlab        = 44
	BlockBrick       = 45
	BlockTNT         = 46
	BlockBookshelf   = 47
	BlockMoss        = 48
	BlockObsidian    = 49

	BlockMaxClassic   = BlockObsidian
	BlockCountClassic = BlockMaxClassic + 1

	BlockCobblestoneSlab = 50
	BlockRope            = 51
	BlockSandstone       = 52
	BlockSnow            = 53
	BlockFire            = 54
	BlockLightPink       = 55
	BlockForestGreen     = 56
	BlockBrown           = 57
	BlockDeepBlue        = 58
	BlockTurquoise       = 59
	BlockIce             = 60
	BlockCeramicTile     = 61
	BlockMagma           = 62
	BlockPillar          = 63
	BlockCrate           = 64
	BlockStoneBrick      = 65

	BlockMaxCPE   = BlockStoneBrick
	BlockCountCPE = BlockMaxCPE + 1

	BlockMax   = 255
	BlockCount = BlockMax + 1
)

var BlockName = [BlockCountCPE]string{
	"air", "stone", "grass", "dirt", "cobblestone", "wood", "sapling",
	"bedrock", "active_water", "water", "active_lava", "lava", "sand",
	"gravel", "gold_ore", "iron_ore", "coal", "log", "leaves", "sponge",
	"glass", "red", "orange", "yellow", "lime", "green", "aqua", "cyan",
	"blue", "purple", "indigo", "violet", "magenta", "pink", "black",
	"gray", "white", "dandelion", "rose", "brown_shroom", "red_shroom",
	"gold", "iron", "doubleslab", "slab", "brick", "tnt", "bookshelf",
	"moss", "obsidian", "cobblestone_slab", "rope", "sandstone", "snow",
	"fire", "light_pink", "forest_green", "brown", "deep_blue", "turquoise",
	"ice", "ceramic_tile", "magma", "pillar", "crate", "stone_brick",
}

// FallbackBlock converts a CPE block to a similar vanilla-compatible one.
func FallbackBlock(block byte) byte {
	switch block {
	case BlockCobblestoneSlab:
		return BlockSlab
	case BlockRope:
		return BlockBrownShroom
	case BlockSandstone:
		return BlockSandstone
	case BlockSnow:
		return BlockAir
	case BlockFire:
		return BlockLava
	case BlockLightPink:
		return BlockPink
	case BlockForestGreen:
		return BlockGreen
	case BlockBrown:
		return BlockDirt
	case BlockDeepBlue:
		return BlockBlue
	case BlockTurquoise:
		return BlockIndigo
	case BlockIce:
		return BlockGlass
	case BlockCeramicTile:
		return BlockIronOre
	case BlockMagma:
		return BlockObsidian
	case BlockPillar:
		return BlockWhite
	case BlockCrate:
		return BlockWood
	case BlockStoneBrick:
		return BlockStone
	default:
		return block
	}
}

const (
	FacePosX = 0
	FaceNegX = 1
	FacePosY = 2
	FaceNegY = 3
	FacePosZ = 4
	FaceNegZ = 5
)

const (
	CollideModeWalk  = 0
	CollideModeSwim  = 1
	CollideModeSolid = 2
)

const (
	WalkSoundNone   = 0
	WalkSoundWood   = 1
	WalkSoundGravel = 2
	WalkSoundGrass  = 3
	WalkSoundStone  = 4
	WalkSoundMetal  = 5
	WalkSoundGlass  = 6
	WalkSoundWool   = 7
	WalkSoundSand   = 8
	WalkSoundSnow   = 9
)

const (
	BlockShapeSprite = 0
	BlockShapeCube   = 16
)

const (
	DrawModeOpaque = 0
	DrawModeGlass  = 1
	DrawModeLeaves = 2
	DrawModeIce    = 3
	DrawModeGas    = 4
)

// BlockDefinition describes a custom block.
type BlockDefinition struct {
	Name     string
	Fallback byte

	Speed       float64
	CollideMode byte
	WalkSound   byte

	BlockLight bool
	FullBright bool
	DrawMode   byte
	Textures   [6]int

	Shape byte
	AABB  AABB

	FogDensity byte
	Fog        color.RGBA
}
