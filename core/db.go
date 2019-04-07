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

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func openDb() {
	var err error
	db, err = sql.Open("sqlite3", "core.sqlite")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS BannedNames(
Name TEXT NOT NULL PRIMARY KEY,
Reason TEXT,
BannedBy TEXT,
Timestamp DATETIME);`)
	if err != nil {
		log.Fatal(err)
	}
}

func BanName(name string, reason string, bannedBy string) {
	_, err := db.Exec(`INSERT INTO BannedNames(Name, Reason, BannedBy, Timestamp)
		VALUES(?, ?, ?, CURRENT_TIMESTAMP)`, name, reason, bannedBy)
	if err != nil {
		log.Fatal(err)
	}
}

func UnbanName(name string) {
	_, err := db.Exec("DELETE FROM BannedNames WHERE Name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
}

func IsNameBanned(name string) (result bool, reason string) {
	rows, err := db.Query("SELECT Reason FROM BannedNames WHERE Name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	result = false
	if rows.Next() {
		result = true
		if err := rows.Scan(&reason); err != nil {
			log.Fatal(err)
		}
	}

	return
}
