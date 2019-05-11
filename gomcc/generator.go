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
	"strconv"
)

type Generator interface {
	Generate(level *Level)
}

type FlatGenerator struct {
	GrassHeight  int
	SurfaceBlock byte
	SoilBlock    byte
}

func newFlatGenerator(args ...string) Generator {
	grassHeight := -1
	if len(args) > 0 {
		grassHeight, _ = strconv.Atoi(args[0])
	}

	return &FlatGenerator{
		GrassHeight:  grassHeight,
		SurfaceBlock: BlockGrass,
		SoilBlock:    BlockDirt,
	}
}

func (generator *FlatGenerator) Generate(level *Level) {
	grassHeight := uint(generator.GrassHeight)
	if generator.GrassHeight < 0 {
		grassHeight = level.height / 2
	}

	for y := uint(0); y < grassHeight; y++ {
		for z := uint(0); z < level.length; z++ {
			for x := uint(0); x < level.width; x++ {
				level.SetBlock(x, y, z, generator.SoilBlock, false)
			}
		}
	}

	for z := uint(0); z < level.length; z++ {
		for x := uint(0); x < level.width; x++ {
			level.SetBlock(x, grassHeight, z, generator.SurfaceBlock, false)
		}
	}
}

var Generators = map[string]func(args ...string) Generator{
	"flat": newFlatGenerator,
}
