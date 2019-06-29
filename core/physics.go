// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *CorePlugin) handlePhysics(level *gomcc.Level, block byte, x, y, z uint) {
	switch block {
	case gomcc.BlockSponge:
		plugin.handleSponge(level, block, x, y, z)
	}
}

func (plugin *CorePlugin) handleSponge(level *gomcc.Level, block byte, x, y, z uint) {
	for yy := y - 2; yy <= y+2; yy++ {
		for zz := z - 2; zz <= z+2; zz++ {
			for xx := x - 2; xx <= x+2; xx++ {
				switch level.GetBlock(xx, yy, zz) {
				case gomcc.BlockWater:
				case gomcc.BlockActiveWater:
					level.SetBlock(xx, yy, zz, gomcc.BlockAir)
				}
			}
		}
	}
}
