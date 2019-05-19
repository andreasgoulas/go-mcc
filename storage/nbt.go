// Copyright (c) 2017-2019 Andrew Goulas
// Licensed under the MIT license.

package storage

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"strings"
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

func NbtMarshal(w io.Writer, name string, v interface{}) error {
	encoder := newNbtEncoder(w)
	return encoder.marshal(name, reflect.ValueOf(v))
}

func NbtUnmarshal(w io.Reader, v interface{}) (err error) {
	decoder := newNbtDecoder(w)
	_, err = decoder.unmarshal(reflect.ValueOf(v).Elem())
	return
}

type nbtEncoder struct {
	w io.Writer
}

func newNbtEncoder(w io.Writer) *nbtEncoder {
	return &nbtEncoder{w}
}

func (nbt *nbtEncoder) writeByte(tag byte) error {
	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeShort(tag int16) error {
	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeInt(tag int32) error {
	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeLong(tag int64) error {
	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeFloat(tag float32) error {
	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeDouble(tag float64) error {
	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeByteArray(tag []byte) error {
	if err := nbt.writeInt(int32(len(tag))); err != nil {
		return err
	}

	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeString(tag string) error {
	if err := nbt.writeShort(int16(len(tag))); err != nil {
		return err
	}

	return binary.Write(nbt.w, binary.BigEndian, []byte(tag))
}

func (nbt *nbtEncoder) writeCompound(v reflect.Value) error {
	for i := 0; i < v.Type().NumField(); i++ {
		field := v.Type().Field(i)
		if field.Anonymous {
			continue
		}

		tag := field.Tag.Get("nbt")
		if tag == "-" {
			continue
		}

		fname := tag
		if fname == "" {
			fname = field.Name
		}

		if err := nbt.marshal(fname, v.Field(i)); err != nil {
			return err
		}
	}

	return nbt.writeByte(NbtTagEnd)
}

func (nbt *nbtEncoder) writeIntArray(tag []int32) error {
	if err := nbt.writeInt(int32(len(tag))); err != nil {
		return err
	}

	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) writeLongArray(tag []int64) error {
	if err := nbt.writeInt(int32(len(tag))); err != nil {
		return err
	}

	return binary.Write(nbt.w, binary.BigEndian, tag)
}

func (nbt *nbtEncoder) marshal(name string, v reflect.Value) (err error) {
	var tagType byte
	switch v.Type().Kind() {
	case reflect.Uint8:
		tagType = NbtTagByte
	case reflect.Int16:
		tagType = NbtTagShort
	case reflect.Int32:
		tagType = NbtTagInt
	case reflect.Int64:
		tagType = NbtTagLong
	case reflect.Float32:
		tagType = NbtTagFloat
	case reflect.Float64:
		tagType = NbtTagDouble
	case reflect.String:
		tagType = NbtTagString
	case reflect.Slice:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			tagType = NbtTagByteArray
		case reflect.Int32:
			tagType = NbtTagIntArray
		case reflect.Int64:
			tagType = NbtTagLongArray
		default:
			return
		}
	case reflect.Struct:
		tagType = NbtTagCompound
	default:
		return
	}

	if err = nbt.writeByte(tagType); err != nil {
		return
	}

	if len(name) != 0 {
		if err = nbt.writeString(name); err != nil {
			return
		}
	}

	switch tagType {
	case NbtTagByte:
		err = nbt.writeByte(byte(v.Uint()))
	case NbtTagShort:
		err = nbt.writeShort(int16(v.Int()))
	case NbtTagInt:
		err = nbt.writeInt(int32(v.Int()))
	case NbtTagLong:
		err = nbt.writeLong(int64(v.Int()))
	case NbtTagFloat:
		err = nbt.writeFloat(float32(v.Float()))
	case NbtTagDouble:
		err = nbt.writeDouble(float64(v.Float()))
	case NbtTagByteArray:
		err = nbt.writeByteArray(v.Bytes())
	case NbtTagString:
		err = nbt.writeString(v.String())
	case NbtTagCompound:
		err = nbt.writeCompound(v)
	case NbtTagIntArray:
		err = nbt.writeIntArray(v.Interface().([]int32))
	case NbtTagLongArray:
		err = nbt.writeLongArray(v.Interface().([]int64))
	}

	return
}

type nbtDecoder struct {
	r io.Reader
}

func newNbtDecoder(r io.Reader) *nbtDecoder {
	return &nbtDecoder{r}
}

func (nbt *nbtDecoder) readByte() (tag byte, err error) {
	err = binary.Read(nbt.r, binary.BigEndian, &tag)
	return
}

func (nbt *nbtDecoder) readShort() (tag int16, err error) {
	err = binary.Read(nbt.r, binary.BigEndian, &tag)
	return
}

func (nbt *nbtDecoder) readInt() (tag int32, err error) {
	err = binary.Read(nbt.r, binary.BigEndian, &tag)
	return
}

func (nbt *nbtDecoder) readLong() (tag int64, err error) {
	err = binary.Read(nbt.r, binary.BigEndian, &tag)
	return
}

func (nbt *nbtDecoder) readFloat() (tag float32, err error) {
	err = binary.Read(nbt.r, binary.BigEndian, &tag)
	return
}

func (nbt *nbtDecoder) readDouble() (tag float64, err error) {
	err = binary.Read(nbt.r, binary.BigEndian, &tag)
	return
}

func (nbt *nbtDecoder) readByteArray() (tag []byte, err error) {
	length, err := nbt.readInt()
	if err != nil {
		return
	}

	tag = make([]byte, length)
	err = binary.Read(nbt.r, binary.BigEndian, tag)
	return
}

func (nbt *nbtDecoder) readString() (tag string, err error) {
	length, err := nbt.readShort()
	if err != nil {
		return
	}

	buf := make([]byte, length)
	if err = binary.Read(nbt.r, binary.BigEndian, buf); err != nil {
		return
	}

	return string(buf), nil
}

func (nbt *nbtDecoder) readCompound(v reflect.Value) (err error) {
	for {
		tagType, err := nbt.unmarshal(v)
		if tagType == NbtTagEnd || err != nil {
			return err
		}
	}

	return
}

func (nbt *nbtDecoder) readIntArray() (tag []int32, err error) {
	length, err := nbt.readInt()
	if err != nil {
		return
	}

	tag = make([]int32, length)
	err = binary.Read(nbt.r, binary.BigEndian, tag)
	return
}

func (nbt *nbtDecoder) readLongArray() (tag []int64, err error) {
	length, err := nbt.readInt()
	if err != nil {
		return
	}

	tag = make([]int64, length)
	err = binary.Read(nbt.r, binary.BigEndian, tag)
	return
}

func (nbt *nbtDecoder) unmarshal(v reflect.Value) (tagType byte, err error) {
	if err = binary.Read(nbt.r, binary.BigEndian, &tagType); err != nil {
		return
	}

	if tagType == NbtTagEnd {
		return
	}

	name, err := nbt.readString()
	if err != nil {
		return
	}

	var target reflect.Value
	name = strings.ToLower(name)
	if v.IsValid() && v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if strings.ToLower(field.Tag.Get("nbt")) == name ||
				(field.Tag.Get("nbt") == "" && strings.ToLower(field.Name) == name) {
				target = v.Field(i)
				break
			}
		}
	}

	var tag interface{}
	switch tagType {
	case NbtTagByte:
		tag, err = nbt.readByte()
	case NbtTagShort:
		tag, err = nbt.readShort()
	case NbtTagInt:
		tag, err = nbt.readInt()
	case NbtTagLong:
		tag, err = nbt.readLong()
	case NbtTagFloat:
		tag, err = nbt.readFloat()
	case NbtTagDouble:
		tag, err = nbt.readDouble()
	case NbtTagByteArray:
		tag, err = nbt.readByteArray()
	case NbtTagString:
		tag, err = nbt.readString()
	case NbtTagCompound:
		err = nbt.readCompound(target)
	case NbtTagIntArray:
		tag, err = nbt.readIntArray()
	case NbtTagLongArray:
		tag, err = nbt.readLongArray()
	default:
		err = errors.New("nbt: invalid tag")
	}

	if err != nil {
		return
	}

	if target.IsValid() {
		if tag != nil {
			target.Set(reflect.ValueOf(tag))
		}
	}

	return
}
