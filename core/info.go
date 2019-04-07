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
	"fmt"
	"strings"
	"time"

	"Go-MCC/gomcc"
)

var commandSeen = gomcc.Command{
	Name:        "seen",
	Description: "Check when a player was last online.",
	Permission:  "core.seen",
	Handler:     handleSeen,
}

func fmtDuration(t time.Duration) string {
	t = t.Round(time.Minute)
	d := t / (24 * time.Hour)
	t -= d * (24 * time.Hour)
	h := t / time.Hour
	t -= h * time.Hour
	m := t / time.Minute
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func handleSeen(sender gomcc.CommandSender, command *gomcc.Command, message string) {
	args := strings.Split(message, " ")
	if len(args) != 1 || len(args[0]) == 0 {
		sender.SendMessage("Usage: " + command.Name + " <player>")
		return
	}

	if client := sender.Server().FindClient(args[0]); client != nil {
		sender.SendMessage("Player " + args[0] + " is currently online")
		return
	}

	lastLogin, err := LastLogin(args[0])
	if err != nil {
		sender.SendMessage("Player " + args[0] + " not found")
		return
	}

	dt := time.Now().Sub(lastLogin)
	sender.SendMessage("Player " + args[0] + " was last seen " + fmtDuration(dt) + " ago")
}