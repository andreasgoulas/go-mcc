// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

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

	level.FillLayers(grassHeight, grassHeight, generator.SurfaceBlock)
	if grassHeight > 0 {
		level.FillLayers(0, grassHeight-1, generator.SoilBlock)
	}
}

var Generators = map[string]func(args ...string) Generator{
	"flat": newFlatGenerator,
}
