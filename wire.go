// Package wire provides an easy and flexible way to serialize and deserialize
// Go structures to binary.
// It has support for arrays, variable length slices and strings, embedded
// structures, and even slices and arrays of embedded structures.
//
// Wire serializes in little endian by default, but this can be overridden with
// the use of struct field tags or by using the WithOrder functions.
// The following tags are supported: big, little, nullterm, sizeof=$
//
//  type Example struct {
//    Cmd         uint8
//    UsernameLen uint16 `wire:"sizeof=Username,big"`
//    Username    string
//    Password    string `wire:"nullterm"`
//  }
//
//  // Note that the value passed in must be a pointer as UsernameLen is modified!
//  wire.Encode(writer, &Example{Cmd: 1, Username: "dajoh", Password: "x"})
package wire

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
)

type sizeofVisitor struct {
	size int
}

type encodeVisitor struct {
	order  binary.ByteOrder
	writer io.Writer
}

type decodeVisitor struct {
	order  binary.ByteOrder
	reader io.Reader
}

// Sizeof returns the size of a value in bytes when serialized.
func Sizeof(v interface{}) (int, error) {
	return sizeof(reflect.ValueOf(v))
}

func sizeof(v reflect.Value) (int, error) {
	vst := sizeofVisitor{}
	err := runVisitor(&vst, v)
	if err != nil {
		return -1, err
	}

	return vst.size, nil
}

func (v *sizeofVisitor) visit(n *node) error {
	switch n.val.Kind() {
	case reflect.Int8, reflect.Uint8:
		v.size++
	case reflect.Int16, reflect.Uint16:
		v.size += 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		v.size += 4
	case reflect.Int64, reflect.Uint64, reflect.Float64, reflect.Complex64:
		v.size += 8
	case reflect.Complex128:
		v.size += 16
	case reflect.Array, reflect.Slice:
		if n.val.Len() > 0 {
			// TODO: this is wrong, should trigger slow path on other variable sized stuff (slice, string, etc)
			if n.val.Type().Elem().Kind() == reflect.Struct {
				for i := 0; i < n.val.Len(); i++ {
					isize, err := sizeof(n.val.Index(i))
					if err != nil {
						return err
					}
					v.size += isize
				}
			} else {
				isize, err := sizeof(n.val.Index(0))
				if err != nil {
					return err
				}
				v.size += n.val.Len() * isize
			}
		}
	case reflect.String:
		if n.nullTerminated {
			v.size += len([]byte(n.val.String())) + 1
		} else {
			v.size += len([]byte(n.val.String()))
		}
	default:
		return errors.New("wire: unsupported type: " + n.val.Kind().String())
	}

	return nil
}

// Encode serializes a value to an io.Writer.
// The value must be a pointer if you use any sizeof fields.
func Encode(w io.Writer, v interface{}) error {
	return encode(w, reflect.ValueOf(v), binary.LittleEndian)
}

// EncodeWithOrder does the same as Encode, but allows you to specify
// the default byte order.
func EncodeWithOrder(w io.Writer, v interface{}, o binary.ByteOrder) error {
	return encode(w, reflect.ValueOf(v), o)
}

func encode(w io.Writer, v reflect.Value, o binary.ByteOrder) error {
	return runVisitor(&encodeVisitor{order: o, writer: w}, v)
}

func (v *encodeVisitor) visit(n *node) error {
	order := v.order
	if n.endianness != nil {
		order = n.endianness
	}

	if n.sizeof.IsValid() {
		switch n.val.Kind() {
		case reflect.Int8, reflect.Int32, reflect.Int64:
			n.val.SetInt(int64(n.sizeof.Len()))
		case reflect.Uint8, reflect.Uint32, reflect.Uint64:
			n.val.SetUint(uint64(n.sizeof.Len()))
		}
	}

	dw := [2]byte{}
	dd := [4]byte{}
	dq := [8]byte{}

	switch n.val.Kind() {
	case reflect.Int8:
		v.writer.Write([]byte{byte(n.val.Int())})
	case reflect.Uint8:
		v.writer.Write([]byte{byte(n.val.Uint())})

	case reflect.Int16:
		order.PutUint16(dw[:], uint16(n.val.Int()))
		v.writer.Write(dw[:])
	case reflect.Uint16:
		order.PutUint16(dw[:], uint16(n.val.Uint()))
		v.writer.Write(dw[:])

	case reflect.Int32:
		order.PutUint32(dd[:], uint32(n.val.Int()))
		v.writer.Write(dd[:])
	case reflect.Uint32:
		order.PutUint32(dd[:], uint32(n.val.Uint()))
		v.writer.Write(dd[:])

	case reflect.Int64:
		order.PutUint64(dq[:], uint64(n.val.Int()))
		v.writer.Write(dq[:])
	case reflect.Uint64:
		order.PutUint64(dq[:], uint64(n.val.Uint()))
		v.writer.Write(dq[:])

	case reflect.Float32:
		order.PutUint32(dd[:], math.Float32bits(float32(n.val.Float())))
		v.writer.Write(dd[:])
	case reflect.Float64:
		order.PutUint64(dq[:], math.Float64bits(n.val.Float()))
		v.writer.Write(dq[:])

	case reflect.Array, reflect.Slice:
		// TODO: fast path for []byte, []int8, []uint8, etc
		for i := 0; i < n.val.Len(); i++ {
			err := encode(v.writer, n.val.Index(i), order)
			if err != nil {
				return err
			}
		}

	case reflect.String:
		io.WriteString(v.writer, n.val.String())
		if n.nullTerminated {
			v.writer.Write([]byte{0x00})
		}

	default:
		return errors.New("wire: unsupported type: " + n.val.Kind().String())
	}

	return nil
}

// Decode deserializes a value from an io.Reader.
// The value must be a pointer.
func Decode(r io.Reader, v interface{}) error {
	return decode(r, reflect.ValueOf(v), binary.LittleEndian)
}

// DecodeWithOrder does the same as decode, but allows you to specify
// the default byte order.
func DecodeWithOrder(r io.Reader, v interface{}, o binary.ByteOrder) error {
	return decode(r, reflect.ValueOf(v), o)
}

func decode(r io.Reader, v reflect.Value, o binary.ByteOrder) error {
	return runVisitor(&decodeVisitor{order: o, reader: r}, v)
}

func (v *decodeVisitor) visit(n *node) error {
	order := v.order
	if n.endianness != nil {
		order = n.endianness
	}

	var err error
	db := [1]byte{}
	dw := [2]byte{}
	dd := [4]byte{}
	dq := [8]byte{}

	switch n.val.Kind() {
	case reflect.Int8:
		_, err = v.reader.Read(db[:])
		n.val.SetInt(int64(db[0]))
	case reflect.Uint8:
		_, err = v.reader.Read(db[:])
		n.val.SetUint(uint64(db[0]))

	case reflect.Int16:
		_, err = v.reader.Read(dw[:])
		n.val.SetInt(int64(order.Uint16(dw[:])))
	case reflect.Uint16:
		_, err = v.reader.Read(dw[:])
		n.val.SetUint(uint64(order.Uint16(dw[:])))

	case reflect.Int32:
		_, err = v.reader.Read(dd[:])
		n.val.SetInt(int64(order.Uint32(dd[:])))
	case reflect.Uint32:
		_, err = v.reader.Read(dd[:])
		n.val.SetUint(uint64(order.Uint32(dd[:])))

	case reflect.Int64:
		_, err = v.reader.Read(dq[:])
		n.val.SetInt(int64(order.Uint64(dq[:])))
	case reflect.Uint64:
		_, err = v.reader.Read(dq[:])
		n.val.SetUint(uint64(order.Uint64(dq[:])))

	case reflect.Array:
		// TODO: fast path for []byte, []int8, []uint8, etc
		for i := 0; i < n.val.Len(); i++ {
			err = decode(v.reader, n.val.Index(i), order)
			if err != nil {
				return err
			}
		}

	case reflect.Slice:
		// TODO: fast path for []byte, []int8, []uint8, etc
		if n.sizeFrom == nil {
			return errors.New("wire: slice with no size source")
		}

		len := int(n.sizeFrom.val.Uint())
		n.val.Set(reflect.MakeSlice(n.val.Type(), len, len))

		for i := 0; i < len; i++ {
			err = decode(v.reader, n.val.Index(i), order)
			if err != nil {
				return err
			}
		}

	case reflect.String:
		if n.nullTerminated {
			var str string
			str, err = readNullTerminatedString(v.reader)
			n.val.SetString(str)
		} else {
			buf := make([]byte, n.sizeFrom.val.Uint())
			_, err = v.reader.Read(buf)
			n.val.SetString(string(buf))
		}

	default:
		return errors.New("wire: unsupported type: " + n.val.Kind().String())
	}

	return err
}

func readNullTerminatedString(r io.Reader) (string, error) {
	buf := []byte{}
	single := []byte{0}

	for {
		_, err := r.Read(single)
		if err != nil {
			return "", err
		} else if single[0] == 0 {
			break
		} else {
			buf = append(buf, single[0])
		}
	}

	return string(buf), nil
}
