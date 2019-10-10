// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package mcc

import (
	"math/rand"
	"strings"
	"unicode"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Location represents the location of an entity in a world.
// Yaw and Pitch are specified in degrees.
type Location struct {
	X, Y, Z, Yaw, Pitch float64
}

// Vector3 represents a three-dimensional integer vector.
type Vector3 struct {
	X, Y, Z int
}

// AABB represents an axis-aligned bounding box.
type AABB struct {
	Min, Max Vector3
}

// RGB represents a 24-bit RGB color.
type RGB struct {
	R, G, B uint8
}

// RGBA represents a 32-bit RGBA color.
type RGBA struct {
	R, G, B, A uint8
}

// NullRGB represents a 24-bit RGB color that may be null.
type NullRGB struct {
	Valid   bool
	R, G, B uint8
}

// IsValidName reports whether name is a valid entity name.
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

// IsValidMessage reports whether message is a valid chat message.
func IsValidMessage(message string) bool {
	for _, c := range message {
		if c > unicode.MaxASCII || !unicode.IsPrint(c) || c == '&' {
			return false
		}
	}

	return true
}

// WordWrap wraps message at width characters.
func WordWrap(message string, width int) (result []string) {
	for _, line := range strings.Split(message, "\n") {
		for {
			if len(line) <= width {
				break
			}

			i := strings.LastIndex(line[:width+1], " ")
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

// RandomUUID returns a random (Version 4) UUID.
func RandomUUID() (uuid [16]byte) {
	rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return
}
