# Go-MCC

## Introduction

Go-MCC is an open source Minecraft classic server written in Go. It is fully
compatible with the original client, World of Minecraft and ClassiCube. It
supports a large subset of the Classic Protocol Extension (CPE) project.

The core functionality of Go-MCC can be extended through the use of plugins. The
Core plugin provides important features typically found in Minecraft servers,
such as ban lists, a rank manager and data persistency.

## Install

```
go get -u github.com/structinf/Go-MCC/gomcc-cli
```

### Core Plugin

```
go get -u -buildmode=plugin github.com/structinf/Go-MCC/core
```

To use a plugin, you need to place it in the `plugins/` directory of the server.
