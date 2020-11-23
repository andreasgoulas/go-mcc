package main

import (
	"math"
	"sync"

	"github.com/structinf/go-mcc/mcc"
)

const (
	maxUpdateQueueLength = math.MaxUint32 / 4
)

func (plugin *plugin) enablePhysics(level *level) {
	sims := []mcc.Simulator{
		&waterSimulator{level: level.Level},
		&lavaSimulator{level: level.Level},
		&sandSimulator{level: level.Level},
	}

	for _, sim := range sims {
		level.AddSimulator(sim)
	}

	level.simulators = append(level.simulators, sims...)
}

func (plugin *plugin) disablePhysics(level *level) {
	for _, sim := range level.simulators {
		level.RemoveSimulator(sim)
	}

	level.simulators = nil
}

type blockUpdate struct {
	index, ticks int
}

type blockUpdateQueue struct {
	lock    sync.Mutex
	updates []blockUpdate
}

func (queue *blockUpdateQueue) Add(index int, delay int) {
	queue.lock.Lock()
	defer queue.lock.Unlock()

	if len(queue.updates) < maxUpdateQueueLength {
		queue.updates = append(queue.updates, blockUpdate{index, delay})
	} else {
		queue.updates = nil
	}
}

func (queue *blockUpdateQueue) Tick() (updates []int) {
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

type waterSimulator struct {
	level *mcc.Level
	queue blockUpdateQueue
}

func (simulator *waterSimulator) Update(block, old byte, index int) {
	if block == mcc.BlockActiveWater || (block == mcc.BlockWater && block == old) {
		simulator.queue.Add(index, 5)
	} else {
		level := simulator.level
		x, y, z := level.Position(index)
		if block == mcc.BlockAir && simulator.checkEdge(x, y, z) {
			if !simulator.checkSponge(x, y, z) {
				level.SetBlock(x, y, z, mcc.BlockActiveWater)
			}
		} else if block != old {
			if block == mcc.BlockSponge {
				simulator.placeSponge(x, y, z)
			} else if old == mcc.BlockSponge {
				simulator.breakSponge(x, y, z)
			}
		}
	}
}

func (simulator *waterSimulator) Tick() {
	level := simulator.level
	for _, index := range simulator.queue.Tick() {
		block := level.Blocks[index]
		if block != mcc.BlockActiveWater && block != mcc.BlockWater {
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

func (simulator *waterSimulator) checkEdge(x, y, z int) bool {
	level := simulator.level
	env := level.EnvConfig
	return (env.EdgeBlock == mcc.BlockActiveWater || env.EdgeBlock == mcc.BlockWater) &&
		y >= (env.EdgeHeight+env.SideOffset) && y < env.EdgeHeight &&
		(x == 0 || z == 0 || x == level.Width-1 || z == level.Length-1)
}

func (simulator *waterSimulator) checkSponge(x, y, z int) bool {
	level := simulator.level
	for yy := max(y-2, 0); yy <= min(y+2, level.Height-1); yy++ {
		for zz := max(z-2, 0); zz <= min(z+2, level.Length-1); zz++ {
			for xx := max(x-2, 0); xx <= min(x+2, level.Width-1); xx++ {
				if level.GetBlock(xx, yy, zz) == mcc.BlockSponge {
					return true
				}
			}
		}
	}

	return false
}

func (simulator *waterSimulator) spread(x, y, z int) {
	level := simulator.level
	switch level.GetBlock(x, y, z) {
	case mcc.BlockAir:
		if !simulator.checkSponge(x, y, z) {
			level.SetBlock(x, y, z, mcc.BlockActiveWater)
		}

	case mcc.BlockActiveLava, mcc.BlockLava:
		level.SetBlock(x, y, z, mcc.BlockStone)
	}
}

func (simulator *waterSimulator) placeSponge(x, y, z int) {
	level := simulator.level
	for yy := max(y-2, 0); yy <= min(y+2, level.Height-1); yy++ {
		for zz := max(z-2, 0); zz <= min(z+2, level.Length-1); zz++ {
			for xx := max(x-2, 0); xx <= min(x+2, level.Width-1); xx++ {
				switch level.GetBlock(xx, yy, zz) {
				case mcc.BlockActiveWater, mcc.BlockWater:
					level.SetBlock(xx, yy, zz, mcc.BlockAir)
				}
			}
		}
	}
}

func (simulator *waterSimulator) breakSponge(x, y, z int) {
	level := simulator.level
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

type lavaSimulator struct {
	level *mcc.Level
	queue blockUpdateQueue
}

func (simulator *lavaSimulator) Update(block, old byte, index int) {
	if block == mcc.BlockActiveLava || (block == mcc.BlockLava && block == old) {
		simulator.queue.Add(index, 30)
	}
}

func (simulator *lavaSimulator) Tick() {
	level := simulator.level
	for _, index := range simulator.queue.Tick() {
		block := level.Blocks[index]
		if block != mcc.BlockActiveLava && block != mcc.BlockLava {
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

func (simulator *lavaSimulator) spread(x, y, z int) {
	level := simulator.level
	switch level.GetBlock(x, y, z) {
	case mcc.BlockAir:
		level.SetBlock(x, y, z, mcc.BlockActiveLava)

	case mcc.BlockActiveWater, mcc.BlockWater:
		level.SetBlock(x, y, z, mcc.BlockStone)
	}
}

type sandSimulator struct {
	level *mcc.Level
}

func (simulator *sandSimulator) Update(block, old byte, index int) {
	if block != mcc.BlockSand && block != mcc.BlockGravel {
		return
	}

	level := simulator.level
	x, y0, z := level.Position(index)
	y1 := y0
	for y1 >= 0 && simulator.check(x, y1-1, z) {
		y1--
	}

	if y0 != y1 {
		level.SetBlock(x, y0, z, mcc.BlockAir)
		level.SetBlock(x, y1, z, block)
	}
}

func (simulator *sandSimulator) Tick() {}

func (simulator *sandSimulator) check(x, y, z int) bool {
	switch simulator.level.GetBlock(x, y, z) {
	case mcc.BlockAir, mcc.BlockActiveWater, mcc.BlockWater,
		mcc.BlockActiveLava, mcc.BlockLava:
		return true
	default:
		return false
	}
}
