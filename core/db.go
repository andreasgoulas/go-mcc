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

const (
	BanTypeName = iota
	BanTypeIp
)

type Database struct {
	conn *sql.DB
}

func newDatabase(path string) *Database {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS BanList(
	Name TEXT NOT NULL PRIMARY KEY,
	Type INTEGER,
	Reason TEXT,
	BannedBy TEXT,
	Timestamp DATETIME);

CREATE TABLE IF NOT EXISTS Players(
	Name TEXT NOT NULL PRIMARY KEY,
	Rank TEXT NOT NULL,
	LastLogin DATETIME);

CREATE TABLE IF NOT EXISTS Ranks(
	Name TEXT PRIMARY KEY);

CREATE TABLE IF NOT EXISTS Permissions(
	Rank TEXT,
	Permission TEXT);`)
	if err != nil {
		log.Fatal(err)
	}

	return &Database{db}
}

func (db *Database) onLogin(name string) {
	_, err := db.conn.Exec(`INSERT OR IGNORE INTO Players(Name, Rank) VALUES(?, "")`, name)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.conn.Exec("UPDATE Players SET LastLogin = CURRENT_TIMESTAMP WHERE Name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *Database) Ban(banType int, name, reason, bannedBy string) {
	_, err := db.conn.Exec(`INSERT OR IGNORE INTO BanList(Name, Type, Reason, BannedBy, Timestamp)
		VALUES(?, ?, ?, ?, CURRENT_TIMESTAMP)`, name, banType, reason, bannedBy)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *Database) Unban(banType int, name string) {
	_, err := db.conn.Exec("DELETE FROM BanList WHERE Name = ? AND Type = ?", name, banType)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *Database) IsBanned(banType int, name string) (result bool, reason string) {
	rows := db.conn.QueryRow("SELECT Reason FROM BanList WHERE Name = ? AND Type = ?", name, banType)
	if err := rows.Scan(&reason); err != nil {
		return
	}

	result = true
	return
}

func (db *Database) SetRank(name, rank string) {
	_, err := db.conn.Exec("UPDATE Players SET Rank = ? WHERE Name = ?", rank, name)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *Database) Rank(name string) (rank string) {
	rows := db.conn.QueryRow("SELECT Rank FROM Players WHERE Name = ?", name)
	if err := rows.Scan(&rank); err != sql.ErrNoRows && err != nil {
		log.Fatal(err)
	}

	return
}

func (db *Database) RankExists(rank string) bool {
	rows, _ := db.conn.Query("SELECT 1 FROM Ranks WHERE Name = ?", rank)
	defer rows.Close()
	return rows.Next()
}

func (db *Database) RankPermissions(rank string) (result []string) {
	rows, err := db.conn.Query(`SELECT Permission FROM Permissions WHERE Rank == ?`, rank)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			log.Fatal(err)
		}

		result = append(result, perm)
	}

	return
}

func (db *Database) PlayerPermissions(name string) []string {
	rank := db.Rank(name)
	if len(rank) == 0 {
		return nil
	}

	return db.RankPermissions(rank)
}

func (db *Database) LastLogin(name string) (lastLogin time.Time, found bool) {
	rows := db.conn.QueryRow("SELECT LastLogin FROM Players WHERE Name = ?", name)
	if rows.Scan(&lastLogin) == nil {
		found = true
	}

	return
}
