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

type LevelGenerator interface {
	Generate(level *Level)
}

var Generators = map[string]LevelGenerator{
	"flat": &FlatGenerator{},
}

type FlatGenerator struct {
	GrassHeight uint
}

func (generator *FlatGenerator) Generate(level *Level) {
	grassHeight := generator.GrassHeight
	if generator.GrassHeight == 0 {
		grassHeight = level.Height / 2
	}

	for y := uint(0); y < grassHeight; y++ {
		for z := uint(0); z < level.Depth; z++ {
			for x := uint(0); x < level.Width; x++ {
				level.SetBlock(x, y, z, BlockGrass, false)
			}
		}
	}
}
