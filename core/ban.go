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

package main

import (
	"sync"
	"time"
)

type BanEntry struct {
	Name      string    `json:"name"`
	Reason    string    `json:"reason"`
	BannedBy  string    `json:"banned-by"`
	Timestamp time.Time `json:"timestamp"`
}

type BanList struct {
	Lock    sync.RWMutex `json:"-"`
	Entries []BanEntry   `json:"entries,omitempty"`
}

func (list *BanList) Ban(name, reason, bannedBy string) bool {
	list.Lock.Lock()
	defer list.Lock.Unlock()

	for _, entry := range list.Entries {
		if entry.Name == name {
			return false
		}
	}

	entry := BanEntry{name, reason, bannedBy, time.Now()}
	list.Entries = append(list.Entries, entry)
	return true
}

func (list *BanList) Unban(name string) bool {
	list.Lock.Lock()
	defer list.Lock.Unlock()

	index := -1
	for i, entry := range list.Entries {
		if entry.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return false
	}

	list.Entries = append(list.Entries[:index], list.Entries[index+1:]...)
	return true
}

func (list *BanList) IsBanned(name string) *BanEntry {
	list.Lock.RLock()
	defer list.Lock.RUnlock()

	for _, entry := range list.Entries {
		if entry.Name == name {
			return &entry
		}
	}

	return nil
}

type BanManager struct {
	IP   BanList `json:"ip,omitempty"`
	Name BanList `json:"name,omitempty"`
}

func (manager *BanManager) Load(path string) {
	manager.IP.Lock.Lock()
	defer manager.IP.Lock.Unlock()

	manager.Name.Lock.Lock()
	defer manager.Name.Lock.Unlock()

	loadJson(path, manager)
}

func (manager *BanManager) Save(path string) {
	manager.IP.Lock.RLock()
	defer manager.IP.Lock.RUnlock()

	manager.Name.Lock.RLock()
	defer manager.Name.Lock.RUnlock()

	saveJson(path, manager)
}
