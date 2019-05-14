// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package storage

import (
	"encoding/binary"
	"io"
)

const (
	NbtTagEnd       = 0
	NbtTagByte      = 1
	NbtTagShort     = 2
	NbtTagInt       = 3
	NbtTagLong      = 4
	NbtTagFloat     = 5
	NbtTagDouble    = 6
	NbtTagByteArray = 7
	NbtTagString    = 8
	NbtTagList      = 9
	NbtTagCompound  = 10
	NbtTagIntArray  = 11
	NbtTagLongArray = 12

	NbtTagMax   = NbtTagLongArray
	NbtTagCount = NbtTagMax + 1
)

type NbtWriter struct {
	w io.Writer
}

func NewNbtWriter(w io.Writer) *NbtWriter {
	return &NbtWriter{w}
}

func (nbt *NbtWriter) tag(tagType byte, name string) {
	binary.Write(nbt.w, binary.BigEndian, tagType)
	if len(name) != 0 {
		binary.Write(nbt.w, binary.BigEndian, struct {
			Length int16
			Value  []byte
		}{int16(len(name)), []byte(name)})
	}
}

func (nbt *NbtWriter) AddCompound(name string) {
	nbt.tag(NbtTagCompound, name)
}

func (nbt *NbtWriter) EndCompound() {
	binary.Write(nbt.w, binary.BigEndian, byte(NbtTagEnd))
}

func (nbt *NbtWriter) AddList(name string, tagType byte, size uint) {
	nbt.tag(NbtTagList, name)
	binary.Write(nbt.w, binary.BigEndian, struct {
		TagType byte
		Length  int32
	}{tagType, int32(size)})
}

func (nbt *NbtWriter) EndList() {
	binary.Write(nbt.w, binary.BigEndian, byte(NbtTagEnd))
}

func (nbt *NbtWriter) AddByte(name string, value byte) {
	nbt.tag(NbtTagByte, name)
	binary.Write(nbt.w, binary.BigEndian, value)
}

func (nbt *NbtWriter) AddShort(name string, value int16) {
	nbt.tag(NbtTagShort, name)
	binary.Write(nbt.w, binary.BigEndian, value)
}

func (nbt *NbtWriter) AddInt(name string, value int32) {
	nbt.tag(NbtTagInt, name)
	binary.Write(nbt.w, binary.BigEndian, value)
}

func (nbt *NbtWriter) AddLong(name string, value int64) {
	nbt.tag(NbtTagLong, name)
	binary.Write(nbt.w, binary.BigEndian, value)
}

func (nbt *NbtWriter) AddFloat(name string, value float32) {
	nbt.tag(NbtTagFloat, name)
	binary.Write(nbt.w, binary.BigEndian, value)
}

func (nbt *NbtWriter) AddDouble(name string, value float64) {
	nbt.tag(NbtTagDouble, name)
	binary.Write(nbt.w, binary.BigEndian, value)
}

func (nbt *NbtWriter) AddString(name string, value string) {
	nbt.tag(NbtTagString, name)
	binary.Write(nbt.w, binary.BigEndian, struct {
		Length int16
		Value  []byte
	}{int16(len(value)), []byte(value)})
}

func (nbt *NbtWriter) AddByteArray(name string, value []byte) {
	nbt.tag(NbtTagByteArray, name)
	binary.Write(nbt.w, binary.BigEndian, struct {
		Length int32
		Value  []byte
	}{int32(len(value)), value})
}

func (nbt *NbtWriter) AddIntArray(name string, value []int32) {
	nbt.tag(NbtTagIntArray, name)
	binary.Write(nbt.w, binary.BigEndian, struct {
		Length int32
		Value  []int32
	}{int32(len(value)), value})
}

func (nbt *NbtWriter) AddLongArray(name string, value []int64) {
	nbt.tag(NbtTagLongArray, name)
	binary.Write(nbt.w, binary.BigEndian, struct {
		Length int32
		Value  []int64
	}{int32(len(value)), value})
}
