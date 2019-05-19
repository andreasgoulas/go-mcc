// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package gomcc

import (
	"math/rand"
	"strings"
	"unicode"
)

// A Location represents the location of an entity in a world.
// Yaw and Pitch are specified in degrees.
type Location struct {
	X, Y, Z, Yaw, Pitch float64
}

type BlockPos struct {
	X, Y, Z uint
}

type AABB struct {
	Min, Max BlockPos
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func IsValidName(name string) bool {
	if len(name) < 3 || len(name) > 16 {
		return false
	}

	for _, c := range name {
		if c > unicode.MaxASCII || (!unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_') {
			return false
		}
	}

	return true
}

func IsValidMessage(message string) bool {
	for _, c := range message {
		if c > unicode.MaxASCII || !unicode.IsPrint(c) || c == '&' {
			return false
		}
	}

	return true
}

func WordWrap(message string, limit int) (result []string) {
	for _, line := range strings.Split(message, "\n") {
		for {
			if len(line) <= limit {
				break
			}

			i := strings.LastIndex(line[:limit+1], " ")
			if i < 0 {
				i = strings.LastIndex(line, " ")
				if i < 0 {
					break
				}
			}

			result = append(result, line[:i])
			line = line[i+1:]
		}

		result = append(result, line)
	}

	return
}

func RandomUUID() (uuid [16]byte) {
	rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return
}
