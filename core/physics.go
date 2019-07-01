// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package main

import (
	"sync"

	"github.com/structinf/Go-MCC/gomcc"
)

func (plugin *CorePlugin) initPhysics(level *gomcc.Level) {
	level.RegisterSimulator(&WaterSimulator{Level: level})
	level.RegisterSimulator(&LavaSimulator{Level: level})
	level.RegisterSimulator(&SandSimulator{Level: level})
}

type blockUpdate struct {
	index, ticks int
}

type BlockUpdateQueue struct {
	lock    sync.Mutex
	updates []blockUpdate
}

func (queue *BlockUpdateQueue) Add(index int, delay int) {
	queue.lock.Lock()
	queue.updates = append(queue.updates, blockUpdate{index, delay})
	queue.lock.Unlock()
}

func (queue *BlockUpdateQueue) Tick() (updates []int) {
	i := 0
	queue.lock.Lock()
	for _, update := range queue.updates {
		update.ticks--
		if update.ticks == 0 {
			updates = append(updates, update.index)
		} else {
			queue.updates[i] = update
			i++
		}
	}
	queue.updates = queue.updates[:i]
	queue.lock.Unlock()
	return
}

type WaterSimulator struct {
	Level *gomcc.Level
	queue BlockUpdateQueue
}

func (simulator *WaterSimulator) Update(block, old byte, index int) {
	if block == gomcc.BlockActiveWater || (block == gomcc.BlockWater && block == old) {
		simulator.queue.Add(index, 5)
	} else if block != old {
		if block == gomcc.BlockSponge {
			simulator.placeSponge(index)
		} else if old == gomcc.BlockSponge {
			simulator.breakSponge(index)
		}
	}
}

func (simulator *WaterSimulator) Tick() {
	level := simulator.Level
	for _, index := range simulator.queue.Tick() {
		block := level.Blocks[index]
		if block != gomcc.BlockActiveWater && block != gomcc.BlockWater {
			return
		}

		x, y, z := level.Position(index)
		if x < level.Width-1 {
			simulator.spread(x+1, y, z)
		}
		if x > 0 {
			simulator.spread(x-1, y, z)
		}
		if z < level.Length-1 {
			simulator.spread(x, y, z+1)
		}
		if z > 0 {
			simulator.spread(x, y, z-1)
		}
		if y > 0 {
			simulator.spread(x, y-1, z)
		}
	}
}

func (simulator *WaterSimulator) spread(x, y, z int) {
	level := simulator.Level
	switch level.GetBlock(x, y, z) {
	case gomcc.BlockAir:
		for yy := max(y-2, 0); yy <= min(y+2, level.Height-1); yy++ {
			for zz := max(z-2, 0); zz <= min(z+2, level.Length-1); zz++ {
				for xx := max(x-2, 0); xx <= min(x+2, level.Width-1); xx++ {
					if level.GetBlock(xx, yy, zz) == gomcc.BlockSponge {
						return
					}
				}
			}
		}

		level.SetBlock(x, y, z, gomcc.BlockActiveWater)

	case gomcc.BlockActiveLava, gomcc.BlockLava:
		level.SetBlock(x, y, z, gomcc.BlockStone)
	}
}

func (simulator *WaterSimulator) placeSponge(index int) {
	level := simulator.Level
	x, y, z := level.Position(index)
	for yy := max(y-2, 0); yy <= min(y+2, level.Height-1); yy++ {
		for zz := max(z-2, 0); zz <= min(z+2, level.Length-1); zz++ {
			for xx := max(x-2, 0); xx <= min(x+2, level.Width-1); xx++ {
				switch level.GetBlock(xx, yy, zz) {
				case gomcc.BlockActiveWater, gomcc.BlockWater:
					level.SetBlock(xx, yy, zz, gomcc.BlockAir)
				}
			}
		}
	}
}

func (simulator *WaterSimulator) breakSponge(index int) {
	level := simulator.Level
	x, y, z := level.Position(index)
	for yy := max(y-3, 0); yy <= min(y+3, level.Height-1); yy++ {
		for zz := max(z-3, 0); zz <= min(z+3, level.Length-1); zz++ {
			for xx := max(x-3, 0); xx <= min(x+3, level.Width-1); xx++ {
				if abs(xx-x) == 3 || abs(yy-y) == 3 || abs(zz-z) == 3 {
					switch level.GetBlock(xx, yy, zz) {
					case gomcc.BlockActiveWater, gomcc.BlockWater:
						level.UpdateBlock(xx, yy, zz)
					}
				}
			}
		}
	}
}

type LavaSimulator struct {
	Level *gomcc.Level
	queue BlockUpdateQueue
}

func (simulator *LavaSimulator) Update(block, old byte, index int) {
	if block == gomcc.BlockActiveLava || (block == gomcc.BlockLava && block == old) {
		simulator.queue.Add(index, 30)
	}
}

func (simulator *LavaSimulator) Tick() {
	level := simulator.Level
	for _, index := range simulator.queue.Tick() {
		block := level.Blocks[index]
		if block != gomcc.BlockActiveLava && block != gomcc.BlockLava {
			return
		}

		x, y, z := level.Position(index)
		if x < level.Width-1 {
			simulator.spread(x+1, y, z)
		}
		if x > 0 {
			simulator.spread(x-1, y, z)
		}
		if z < level.Length-1 {
			simulator.spread(x, y, z+1)
		}
		if z > 0 {
			simulator.spread(x, y, z-1)
		}
		if y > 0 {
			simulator.spread(x, y-1, z)
		}
	}
}

func (simulator *LavaSimulator) spread(x, y, z int) {
	level := simulator.Level
	switch level.GetBlock(x, y, z) {
	case gomcc.BlockAir:
		level.SetBlock(x, y, z, gomcc.BlockActiveLava)

	case gomcc.BlockActiveWater, gomcc.BlockWater:
		level.SetBlock(x, y, z, gomcc.BlockStone)
	}
}

type SandSimulator struct {
	Level *gomcc.Level
}

func (simulator *SandSimulator) Update(block, old byte, index int) {
	if block != gomcc.BlockSand && block != gomcc.BlockGravel {
		return
	}

	level := simulator.Level
	x, y0, z := level.Position(index)
	y1 := y0
	for y1 >= 0 && simulator.check(x, y1-1, z) {
		y1--
	}

	if y0 != y1 {
		level.SetBlock(x, y0, z, gomcc.BlockAir)
		level.SetBlock(x, y1, z, block)
	}
}

func (simulator *SandSimulator) Tick() {}

func (simulator *SandSimulator) check(x, y, z int) bool {
	switch simulator.Level.GetBlock(x, y, z) {
	case gomcc.BlockAir, gomcc.BlockActiveWater, gomcc.BlockWater,
		gomcc.BlockActiveLava, gomcc.BlockLava:
		return true
	default:
		return false
	}
}
