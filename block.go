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

type BlockID byte

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

	BlockMax   = BlockObsidian
	BlockCount = BlockMax + 1

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
)

func FallbackBlock(block BlockID) BlockID {
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

var BlockInfo = [BlockCountCPE]struct {
	Name string
}{
	{"air"},
	{"stone"},
	{"grass"},
	{"dirt"},
	{"cobblestone"},
	{"wood"},
	{"sapling"},
	{"bedrock"},
	{"active_water"},
	{"water"},
	{"active_lava"},
	{"lava"},
	{"sand"},
	{"gravel"},
	{"gold_ore"},
	{"iron_ore"},
	{"coal"},
	{"log"},
	{"leaves"},
	{"sponge"},
	{"glass"},
	{"red"},
	{"orange"},
	{"yellow"},
	{"lime"},
	{"green"},
	{"aqua"},
	{"cyan"},
	{"blue"},
	{"purple"},
	{"indigo"},
	{"violet"},
	{"magenta"},
	{"pink"},
	{"black"},
	{"gray"},
	{"white"},
	{"dandelion"},
	{"rose"},
	{"brown_shroom"},
	{"red_shroom"},
	{"gold"},
	{"iron"},
	{"doubleslab"},
	{"slab"},
	{"brick"},
	{"tnt"},
	{"bookshelf"},
	{"moss"},
	{"obsidian"},
	{"cobblestone_slab"},
	{"rope"},
	{"sandstone"},
	{"snow"},
	{"fire"},
	{"light_pink"},
	{"forest_green"},
	{"brown"},
	{"deep_blue"},
	{"turquoise"},
	{"ice"},
	{"ceramic_tile"},
	{"magma"},
	{"pillar"},
	{"crate"},
	{"stone_brick"},
}
