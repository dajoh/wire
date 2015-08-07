package wire

import (
	"encoding/binary"
	"errors"
	"reflect"
	"regexp"
)

type node struct {
	val            reflect.Value
	sizeof         reflect.Value
	sizeFrom       *node
	sizeFroms      map[string]*node
	endianness     binary.ByteOrder
	nullTerminated bool
}

type visitor interface {
	visit(*node) error
}

var tagRegexp = regexp.MustCompile("big|little|nullterm|(sizeof)=(\\w+)")

func runVisitor(v visitor, val reflect.Value) error {
	return runVisitorInternal(v, val, nil, nil)
}

func runVisitorInternal(v visitor, val reflect.Value, p *node, f *reflect.StructField) error {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	n := &node{
		val: val,
	}

	if p != nil && p.sizeFroms != nil {
		n.sizeFrom = p.sizeFroms[f.Name]
	}

	if f != nil {
		tag := f.Tag.Get("wire")
		for _, x := range tagRegexp.FindAllStringSubmatch(tag, -1) {
			if x[0] == "big" {
				n.endianness = binary.BigEndian
			} else if x[0] == "little" {
				n.endianness = binary.LittleEndian
			} else if x[0] == "nullterm" {
				n.nullTerminated = true
			} else if x[1] == "sizeof" {
				n.sizeof = p.val.FieldByName(x[2])
				if p.sizeFroms == nil {
					p.sizeFroms = make(map[string]*node)
				}
				p.sizeFroms[x[2]] = n
			}
		}
	}

	switch val.Kind() {
	case
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Slice, reflect.String:
		return v.visit(n)
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			fld := val.Type().Field(i)
			err := runVisitorInternal(v, val.Field(i), n, &fld)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return errors.New("wire: unsupported type: " + val.Kind().String())
}
