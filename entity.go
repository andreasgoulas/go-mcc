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

import (
	"time"
)

type Location struct {
	X, Y, Z, Yaw, Pitch float64
}

type Entity interface {
	GetName() string
	GetID() byte
	SetID(id byte)
	GetLocation() Location
	GetLevel() *Level
	Teleport(location Location)
	TeleportLevel(level *Level)
	Update(dt time.Duration)
}
