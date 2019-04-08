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
	"strings"
	"unicode"
)

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
