package mcc

import (
	"math"
	"sync"
)

const (
	maxUpdateQueueLength = math.MaxUint32 / 4
)

type blockUpdate struct {
	index, ticks int
}

type blockUpdateQueue struct {
	lock    sync.Mutex
	updates []blockUpdate
}

func (queue *blockUpdateQueue) add(index int, delay int) {
	queue.lock.Lock()
	defer queue.lock.Unlock()

	if len(queue.updates) < maxUpdateQueueLength {
		queue.updates = append(queue.updates, blockUpdate{index, delay})
	} else {
		queue.updates = nil
	}
}

func (queue *blockUpdateQueue) tick() (updates []int) {
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

// WaterSimulator is an implementation of the Simulator interface that handles
// water and sponge physics.
type WaterSimulator struct {
	Level *Level
	queue blockUpdateQueue
}

// Update implements Simulator.
func (simulator *WaterSimulator) Update(block, old byte, index int) {
	if block == BlockActiveWater || (block == BlockWater && block == old) {
		simulator.queue.add(index, 5)
	} else {
		level := simulator.Level
		x, y, z := level.Position(index)
		if block == BlockAir && simulator.checkEdge(x, y, z) {
			if !simulator.checkSponge(x, y, z) {
				level.SetBlock(x, y, z, BlockActiveWater)
			}
		} else if block != old {
			if block == BlockSponge {
				simulator.placeSponge(x, y, z)
			} else if old == BlockSponge {
				simulator.breakSponge(x, y, z)
			}
		}
	}
}

// Tick implements Simulator.
func (simulator *WaterSimulator) Tick() {
	level := simulator.Level
	for _, index := range simulator.queue.tick() {
		block := level.Blocks[index]
		if block != BlockActiveWater && block != BlockWater {
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

func (simulator *WaterSimulator) checkEdge(x, y, z int) bool {
	level := simulator.Level
	env := level.EnvConfig
	return (env.EdgeBlock == BlockActiveWater || env.EdgeBlock == BlockWater) &&
		y >= (env.EdgeHeight+env.SideOffset) && y < env.EdgeHeight &&
		(x == 0 || z == 0 || x == level.Width-1 || z == level.Length-1)
}

func (simulator *WaterSimulator) checkSponge(x, y, z int) bool {
	level := simulator.Level
	for yy := max(y-2, 0); yy <= min(y+2, level.Height-1); yy++ {
		for zz := max(z-2, 0); zz <= min(z+2, level.Length-1); zz++ {
			for xx := max(x-2, 0); xx <= min(x+2, level.Width-1); xx++ {
				if level.GetBlock(xx, yy, zz) == BlockSponge {
					return true
				}
			}
		}
	}

	return false
}

func (simulator *WaterSimulator) spread(x, y, z int) {
	level := simulator.Level
	switch level.GetBlock(x, y, z) {
	case BlockAir:
		if !simulator.checkSponge(x, y, z) {
			level.SetBlock(x, y, z, BlockActiveWater)
		}

	case BlockActiveLava, BlockLava:
		level.SetBlock(x, y, z, BlockStone)
	}
}

func (simulator *WaterSimulator) placeSponge(x, y, z int) {
	level := simulator.Level
	for yy := max(y-2, 0); yy <= min(y+2, level.Height-1); yy++ {
		for zz := max(z-2, 0); zz <= min(z+2, level.Length-1); zz++ {
			for xx := max(x-2, 0); xx <= min(x+2, level.Width-1); xx++ {
				switch level.GetBlock(xx, yy, zz) {
				case BlockActiveWater, BlockWater:
					level.SetBlock(xx, yy, zz, BlockAir)
				}
			}
		}
	}
}

func (simulator *WaterSimulator) breakSponge(x, y, z int) {
	level := simulator.Level
	for yy := max(y-3, 0); yy <= min(y+3, level.Height-1); yy++ {
		for zz := max(z-3, 0); zz <= min(z+3, level.Length-1); zz++ {
			for xx := max(x-3, 0); xx <= min(x+3, level.Width-1); xx++ {
				index := level.Index(xx, yy, zz)
				block := level.Blocks[index]
				simulator.Update(block, block, index)
			}
		}
	}
}

// LavaSimulator is an implementation of the Simulator interface that handles
// lava physics.
type LavaSimulator struct {
	Level *Level
	queue blockUpdateQueue
}

// Update implements Simulator.
func (simulator *LavaSimulator) Update(block, old byte, index int) {
	if block == BlockActiveLava || (block == BlockLava && block == old) {
		simulator.queue.add(index, 30)
	}
}

// Tick implements Simulator.
func (simulator *LavaSimulator) Tick() {
	level := simulator.Level
	for _, index := range simulator.queue.tick() {
		block := level.Blocks[index]
		if block != BlockActiveLava && block != BlockLava {
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
	case BlockAir:
		level.SetBlock(x, y, z, BlockActiveLava)

	case BlockActiveWater, BlockWater:
		level.SetBlock(x, y, z, BlockStone)
	}
}

// SandSimulator is an implementation of the Simulator interface that handles
// falling block physics.
type SandSimulator struct {
	Level *Level
}

// Update implements Simulator.
func (simulator *SandSimulator) Update(block, old byte, index int) {
	if block != BlockSand && block != BlockGravel {
		return
	}

	level := simulator.Level
	x, y0, z := level.Position(index)
	y1 := y0
	for y1 >= 0 && simulator.check(x, y1-1, z) {
		y1--
	}

	if y0 != y1 {
		level.SetBlock(x, y0, z, BlockAir)
		level.SetBlock(x, y1, z, block)
	}
}

// Tick implements Simulator.
func (simulator *SandSimulator) Tick() {}

func (simulator *SandSimulator) check(x, y, z int) bool {
	switch simulator.Level.GetBlock(x, y, z) {
	case BlockAir, BlockActiveWater, BlockWater,
		BlockActiveLava, BlockLava:
		return true
	default:
		return false
	}
}
