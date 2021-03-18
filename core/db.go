package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const dbSchema = `
CREATE TABLE banned_names(
	name TEXT PRIMARY KEY,
	reason TEXT,
	banned_by TEXT,
	timestamp DATETIME
);

CREATE TABLE banned_ips(
	ip TEXT PRIMARY KEY,
	reason TEXT,
	banned_by TEXT,
	timestamp DATETIME
);

CREATE TABLE levels(
	name TEXT PRIMARY KEY,
	motd TEXT NOT NULL,
	physics INTEGER NOT NULL
);

CREATE TABLE players(
	name TEXT PRIMARY KEY,
	rank TEXT,
	first_login DATETIME,
	last_login DATETIME,
	nickname TEXT NOT NULL,
	ignore_list TEXT NOT NULL,
	mute INTEGER NOT NULL
);

CREATE TABLE ranks(
	name TEXT PRIMARY KEY,
	tag TEXT,
	permissions INTEGER NOT NULL
);

CREATE TABLE command_rules(
	command TEXT NOT NULL,
	rank TEXT NOT NULL,
	access INTEGER NOT NULL,
	PRIMARY KEY (command, rank)
);

CREATE TABLE block_rules(
	block_id INTEGER NOT NULL,
	action INTEGER NOT NULL,
	rank TEXT NOT NULL,
	access INTEGER NOT NULL,
	PRIMARY KEY (block_id, action, rank)
);

CREATE TABLE config(
	cfg_key TEXT PRIMARY KEY,
	cfg_value TEXT NOT NULL
);

INSERT INTO ranks(name, permissions)
VALUES("op", 0xffffffff);

INSERT INTO config(cfg_key, cfg_value)
VALUES("default_rank", "");
`

type dbLevel struct {
	MOTD    string `db:"motd"`
	Physics bool   `db:"physics"`
}

type dbPlayer struct {
	Rank       sql.NullString `db:"rank"`
	FirstLogin time.Time      `db:"first_login"`
	LastLogin  time.Time      `db:"last_login"`
	Nickname   string         `db:"nickname"`
	IgnoreList string         `db:"ignore_list"`
	Mute       bool           `db:"mute"`
}

type dbRank struct {
	Name        string         `db:"name"`
	Tag         sql.NullString `db:"tag"`
	Permissions uint32         `db:"permissions"`
}

type dbCommandRule struct {
	Command string `db:"command"`
	Rank    string `db:"rank"`
	Access  bool   `db:"access"`
}

type dbBlockRule struct {
	BlockID int    `db:"block_id"`
	Action  int    `db:"action"`
	Rank    string `db:"rank"`
	Access  bool   `db:"access"`
}

type db struct {
	*sqlx.DB
}

func newDb(path string) *db {
	pdb, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		log.Println(err)
		return nil
	}

	var version int
	pdb.Get(&version, "PRAGMA schema_version")
	if version == 0 {
		pdb.MustExec(dbSchema)
	}

	return &db{DB: pdb}
}

func (db *db) ban(name, reason, banned_by string) {
	db.MustExec(`
REPLACE INTO banned_names(name, reason, banned_by, timestamp)
VALUES(?, ?, ?, CURRENT_TIMESTAMP)`, name, reason, banned_by)
}

func (db *db) banIP(ip, reason, banned_by string) {
	db.MustExec(`
REPLACE INTO banned_ips(ip, reason, banned_by, timestamp)
VALUES(?, ?, ?, CURRENT_TIMESTAMP)`, ip, reason, banned_by)
}

func (db *db) unban(name string) bool {
	r := db.MustExec("DELETE FROM banned_names WHERE name = ?", name)
	rows, _ := r.RowsAffected()
	return rows > 0
}

func (db *db) unbanIP(ip string) bool {
	r := db.MustExec("DELETE FROM banned_ips WHERE ip = ?", ip)
	rows, _ := r.RowsAffected()
	return rows > 0
}

func (db *db) checkBan(addr, name string) (bool, string) {
	var reason sql.NullString
	err := db.Get(&reason, `
SELECT reason FROM banned_ips WHERE ip = ? UNION
SELECT reason FROM banned_names WHERE name = ?`, addr, name)
	return err != sql.ErrNoRows, reason.String
}

func (db *db) queryPlayer(name string) (player dbPlayer, ok bool) {
	ok = db.Get(&player, `
SELECT rank, first_login, last_login, nickname,
ignore_list, mute FROM players WHERE name = ?`, name) != sql.ErrNoRows
	return
}

func (db *db) updatePlayer(name string, player *dbPlayer) {
	db.MustExec(`
REPLACE INTO players(name, rank, first_login, last_login,
nickname, ignore_list, mute) VALUES(?, ?, ?, ?, ?, ?, ?)`,
		name, player.Rank, player.FirstLogin, player.LastLogin,
		player.Nickname, player.IgnoreList, player.Mute)
}

func (db *db) queryLevel(name string) (level dbLevel, ok bool) {
	ok = db.Get(&level, `
SELECT motd, physics FROM levels WHERE name = ?`, name) != sql.ErrNoRows
	return
}

func (db *db) updateLevel(name string, level *dbLevel) {
	db.MustExec("REPLACE INTO levels(name, motd, physics) VALUES(?, ?, ?)",
		name, level.MOTD, level.Physics)
}

func (db *db) queryRanks() (ranks []dbRank) {
	db.Select(&ranks, "SELECT name, tag, permissions FROM ranks")
	return
}

func (db *db) queryCommandRules() (rules []dbCommandRule) {
	db.Select(&rules, "SELECT command, rank, access FROM command_rules")
	return
}

func (db *db) queryBlockRules() (rules []dbBlockRule) {
	db.Select(&rules, "SELECT block_id, action, rank, access FROM block_rules")
	return
}

func (db *db) queryConfig(key string) (value string) {
	db.Get(&value, "SELECT cfg_value FROM config WHERE cfg_key = ?", key)
	return
}
