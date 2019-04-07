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
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func dbOpen() {
	var err error
	db, err = sql.Open("sqlite3", "core.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS BanList(
	Name TEXT NOT NULL PRIMARY KEY,
	Type TEXT NOT NULL,
	Reason TEXT,
	BannedBy TEXT,
	Timestamp DATETIME);

CREATE TABLE IF NOT EXISTS Players(
	Name TEXT NOT NULL PRIMARY KEY,
	LastLogin DATETIME)`)
	if err != nil {
		log.Fatal(err)
	}
}

func dbOnLogin(name string) {
	_, err := db.Exec("INSERT OR IGNORE INTO Players(Name) VALUES(?)", name)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("UPDATE Players SET LastLogin = CURRENT_TIMESTAMP WHERE Name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
}

const (
	BanTypeName = "Name"
	BanTypeIp   = "IP"
)

func Ban(banType string, name string, reason string, bannedBy string) error {
	_, err := db.Exec(`INSERT INTO BanList(Name, Type, Reason, BannedBy, Timestamp)
		VALUES(?, ?, ?, ?, CURRENT_TIMESTAMP)`, name, banType, reason, bannedBy)
	return err
}

func Unban(banType string, name string) {
	_, err := db.Exec("DELETE FROM BanList WHERE Name = ? AND Type = ?", name, banType)
	if err != nil {
		log.Fatal(err)
	}
}

func IsBanned(banType string, name string) (result bool, reason string) {
	rows := db.QueryRow("SELECT Reason FROM BanList WHERE Name = ? AND Type = ?", name, banType)
	if err := rows.Scan(&reason); err != nil {
		return
	}

	result = true
	return
}

func LastLogin(name string) (lastLogin time.Time, err error) {
	rows := db.QueryRow("SELECT LastLogin FROM Players WHERE Name = ?", name)
	err = rows.Scan(&lastLogin)
	return
}
