# go-mcc

## Introduction

go-mcc is an open source Minecraft classic server written in Go. It is fully
compatible with the original client, World of Minecraft and ClassiCube. It
supports a large subset of the Classic Protocol Extension (CPE) project.

The core functionality of go-mcc can be extended through the use of plugins. The
Core plugin provides important features typically found in Minecraft servers,
such as ban lists and data persistence. The configuration and data are stored in
an SQLite database.

go-mcc implements a rank-based permission system. Users can execute only those
commands for which they have the required permissions, which are defined by one
or more permission flags. The use of a command can also be explicitly allowed
or denied for each rank.

## Install

### Building from source

The following commands will build the server executable `go-mcc` and the Core
plugin `plugins/core.so`.

```
git clone git@github.com:AndreasGoulas/go-mcc.git
cd go-mcc
make all
```

To use a plugin, you need to place it in the `plugins/` directory of the server.

## Configuration

The server can be configured via the `server.json` file.

Field       |Type   |Description
------------|-------|----------------------------------------------------------
server-port |integer|Port the server is listening on.
server-name |string |Name of the server.
motd        |string |Message of the day displayed when players join the server.
verify-names|boolean|Whether to verify the player names.
public      |boolean|Whether the server should be displayed on the server list.
max-players |integer|Maximum number of players connected at the same time.
heartbeat   |string |Heartbeat URL.
main-level  |string |Name of the main level.

Core can be configured using SQL. `core.db` is created the first time that the
server runs. The following tables can be edited to configure the player
permissions.

### ranks

This table stores the rank entries.

Field      |Type   |Description
-----------|-------|-------------------------------------------------------
name       |string |Rank name.
tag        |string |Tag that prefixes the name of all members of this rank.
permissions|integer|Bitwise-OR of one or more permission flags.

Core defines the following permission flags.

Name    |Value|Commands
--------|-----|----------------------------------------------
operator|1    |/stop, /rank, /skin
ban     |2    |/ban, /banip, /unban, /unbanip
kick    |4    |/kick
chat    |8    |/mute, /nick, /say
teleport|16   |/tp
summon  |32   |/summon
level   |64   |/env, /load, /main, /newlvl, /physics, /save...

The `op` rank, which has access to all commands, is created by default.

### command_rules

This table stores the explicit command permissions for each rank.

Field  |Type   |Description
-------|-------|-----------------------------------------
command|string |Command name.
rank   |string |Rank name.
access |integer|Whether the command is allowed or denied.

### block_rules

This table stores the block permissions for each rank.

Field   |Type   |Description
--------|-------|-----------------------------------------
block_id|integer|Block ID.
action  |integer|Break = 0, Place = 1
rank    |string |Rank name.
access  |integer|Whether the action is allowed or denied.

### config

This table stores the plugin configuration options.

Field    |Type
---------|------
cfg_key  |string
cfg_value|string

The following options are supported.

Key         |Description
------------|---------------------
default_rank|Name of default rank.

## License

go-mcc is licensed under the [MIT License](https://opensource.org/licenses/MIT).
