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

package core

import (
	"sync"
)

type Rank struct {
	Permissions []string `json:"permissions,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Suffix      string   `json:"suffix,omitempty"`
}

type RankManager struct {
	Lock    sync.RWMutex     `json:"-"`
	Ranks   map[string]*Rank `json:"ranks"`
	Default string           `json:"default"`
}

func (manager *RankManager) Load(path string) {
	manager.Lock.Lock()
	loadJson(path, manager)
	manager.Lock.Unlock()
}

func (manager *RankManager) Save(path string) {
	manager.Lock.Lock()
	saveJson(path, manager)
	manager.Lock.Unlock()
}

func (manager *RankManager) Rank(name string) (rank *Rank, ok bool) {
	manager.Lock.Lock()
	defer manager.Lock.Unlock()
	rank, ok = manager.Ranks[name]
	return
}
