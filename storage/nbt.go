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

// NbtMarshal writes the NBT representation of v into w.
// name specifies the name of the root tag.
func NbtMarshal(w io.Writer, name string, v interface{}) error {
	encoder := newNbtEncoder(w)
	return encoder.writeTag(name, reflect.ValueOf(v))
}

// NbtUnmarshal reads NBT data from r into v.
func NbtUnmarshal(r io.Reader, v interface{}) error {
	decoder := newNbtDecoder(r)
	_, err := decoder.readTag(reflect.ValueOf(v).Elem())
	return err
}

type nbtEncoder struct {
	w io.Writer
}

func newNbtEncoder(w io.Writer) *nbtEncoder {
	return &nbtEncoder{w}
}

func (nbt *nbtEncoder) tagType(t reflect.Type) byte {
	switch t.Kind() {
	case reflect.Uint8:
		return NbtTagByte
	case reflect.Int16:
		return NbtTagShort
	case reflect.Int32:
		return NbtTagInt
	case reflect.Int64:
		return NbtTagLong
	case reflect.Float32:
		return NbtTagFloat
	case reflect.Float64:
		return NbtTagDouble
	case reflect.String:
		return NbtTagString
	case reflect.Slice:
		switch t.Elem().Kind() {
		case reflect.Uint8:
			return NbtTagByteArray
		case reflect.Int32:
			return NbtTagIntArray
		case reflect.Int64:
			return NbtTagLongArray
		default:
			return NbtTagList
		}
	case reflect.Map, reflect.Struct:
		return NbtTagCompound
	}

	return NbtTagEnd
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

func (nbt *nbtEncoder) writeList(v reflect.Value) (err error) {
	tagType := nbt.tagType(v.Type().Elem())
	if tagType == NbtTagEnd {
		return errors.New("nbt: invalid type")
	}

	if err = nbt.writeByte(tagType); err != nil {
		return
	}

	if err = nbt.writeInt(int32(v.Len())); err != nil {
		return
	}

	for i := 0; i < v.Len(); i++ {
		if err = nbt.writePayload(tagType, v.Index(i)); err != nil {
			return
		}
	}

	return
}

func (nbt *nbtEncoder) writeCompound(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return errors.New("nbt: invalid type")
		}

		if v.IsNil() {
			break
		}

		iter := v.MapRange()
		for iter.Next() {
			if err := nbt.writeTag(iter.Key().String(), iter.Value()); err != nil {
				return err
			}
		}

	case reflect.Struct:
		for i := 0; i < v.Type().NumField(); i++ {
			field := v.Type().Field(i)
			if field.Anonymous {
				if field.Type.Kind() == reflect.Map {
					if err := nbt.writeCompound(v.Field(i)); err != nil {
						return err
					}
				}

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

			if err := nbt.writeTag(fname, v.Field(i)); err != nil {
				return err
			}
		}
	}

	return nil
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

func (nbt *nbtEncoder) writePayload(tagType byte, v reflect.Value) (err error) {
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
	case NbtTagList:
		err = nbt.writeList(v)
	case NbtTagCompound:
		if err = nbt.writeCompound(v); err != nil {
			return
		}

		err = nbt.writeByte(NbtTagEnd)
	case NbtTagIntArray:
		err = nbt.writeIntArray(v.Interface().([]int32))
	case NbtTagLongArray:
		err = nbt.writeLongArray(v.Interface().([]int64))
	}

	return
}

func (nbt *nbtEncoder) writeTag(name string, v reflect.Value) (err error) {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	tagType := nbt.tagType(v.Type())
	if tagType == NbtTagEnd {
		return errors.New("nbt: invalid type")
	}

	if err = nbt.writeByte(tagType); err != nil {
		return
	}

	if err = nbt.writeString(name); err != nil {
		return
	}

	return nbt.writePayload(tagType, v)
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

func (nbt *nbtDecoder) readList(v reflect.Value) (out reflect.Value, err error) {
	tagType, err := nbt.readByte()
	if err != nil {
		return
	}

	length, err := nbt.readInt()
	if err != nil {
		return
	}

	for i := int32(0); i < length; i++ {
		tmp := reflect.Indirect(reflect.New(v.Type().Elem()))
		if err = nbt.readPayload(tagType, tmp); err != nil {
			return
		}

		if v.IsValid() {
			v = reflect.Append(v, tmp)
		}
	}

	return v, nil
}

func (nbt *nbtDecoder) readCompound(v reflect.Value) (err error) {
	switch v.Kind() {
	case reflect.Interface:
		if v.NumMethod() != 0 {
			return errors.New("nbt: invalid type")
		}

		var m map[string]interface{}
		v.Set(reflect.MakeMap(reflect.TypeOf(m)))
		v = v.Elem()

	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return errors.New("nbt: invalid type")
		}

		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	}

	for {
		tagType, err := nbt.readTag(v)
		if tagType == NbtTagEnd || err != nil {
			return err
		}
	}
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

func (nbt *nbtDecoder) readPayload(tagType byte, v reflect.Value) (err error) {
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
	case NbtTagList:
		if list, err := nbt.readList(v); err == nil && v.IsValid() {
			v.Set(list)
		}

		return
	case NbtTagCompound:
		err = nbt.readCompound(v)
		return
	case NbtTagIntArray:
		tag, err = nbt.readIntArray()
	case NbtTagLongArray:
		tag, err = nbt.readLongArray()
	default:
		err = errors.New("nbt: invalid tag")
	}

	if err == nil && v.IsValid() {
		v.Set(reflect.ValueOf(tag))
	}

	return
}

func (nbt *nbtDecoder) readTag(v reflect.Value) (tagType byte, err error) {
	if tagType, err = nbt.readByte(); err != nil {
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
	switch v.Kind() {
	case reflect.Struct:
		key := strings.ToLower(name)
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			fname := strings.ToLower(field.Name)
			tag := strings.ToLower(field.Tag.Get("nbt"))
			if tag == key || (tag == "" && fname == key) {
				target = v.Field(i)
				break
			}
		}

		if target.IsValid() {
			break
		}

		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.Anonymous && field.Type.Kind() == reflect.Map {
				target = v.Field(i)
				break
			}
		}

		if !target.IsValid() {
			break
		}

		v = target
		if v.Type().Key().Kind() != reflect.String {
			err = errors.New("nbt: invalid type")
			return
		}

		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}

		fallthrough

	case reflect.Map:
		mapElem := v.Type().Elem()
		target = reflect.New(mapElem).Elem()
	}

	err = nbt.readPayload(tagType, target)
	if v.Kind() == reflect.Map {
		v.SetMapIndex(reflect.ValueOf(name), target)
	}

	return
}
