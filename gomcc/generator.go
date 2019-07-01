// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

import (
	"strconv"
)

// Generator is the interface that must be implemented by level generators.
type Generator interface {
	Generate(level *Level)
}

// FlatGenerator is an implementation of the Generator interface that can
// generate flat grass levels.
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

// Generate implements Generator.
func (generator *FlatGenerator) Generate(level *Level) {
	grassHeight := generator.GrassHeight
	if grassHeight < 0 {
		grassHeight = level.Height / 2
	}

	level.FillLayers(grassHeight, grassHeight, generator.SurfaceBlock)
	if grassHeight > 0 {
		level.FillLayers(0, grassHeight-1, generator.SoilBlock)
	}
}

var Generators = map[string]func(args ...string) Generator{
	"flat": newFlatGenerator,
}
