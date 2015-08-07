package wire

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"testing"
)

type innerStruct struct {
	U32 uint32
}

type testStruct struct {
	I8  int8
	I16 int16
	I32 int32 `wire:"little"`
	I64 int64

	U8  uint8
	U16 uint16
	U32 uint32 `wire:"big"`
	U64 uint64

	AU32 [4]uint32
	SU32 [4]uint32

	TF uint32 `wire:"little,sizeof=SIS"`
	IS innerStruct

	AIS [2]innerStruct
	SIS []innerStruct

	SZ string `wire:"nullterm"`
	SS uint32 `wire:"sizeof=SQ"`
	SQ string
}

var refStruct = testStruct{
	I8:  0x11,
	I16: 0x1122,
	I32: 0x11223344,         // little
	I64: 0x1122334455667788, // 15

	U8:  0x11,
	U16: 0x1122,
	U32: 0x11223344,
	U64: 0x1122334455667788, // 30

	AU32: [4]uint32{0, 1, 2, 3},
	SU32: [4]uint32{0, 1, 2, 3}, // 62

	TF: 0,                            // little
	IS: innerStruct{U32: 0x11223344}, // 70

	AIS: [2]innerStruct{{U32: 0}, {U32: 1}},
	SIS: []innerStruct{{U32: 0}, {U32: 1}}, // 86

	SZ: "hello",
	SS: 0,
	SQ: "banan", // 101
}

var refBytes = []byte{
	0x11,
	0x11, 0x22,
	0x44, 0x33, 0x22, 0x11,
	0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,

	0x11,
	0x11, 0x22,
	0x11, 0x22, 0x33, 0x44,
	0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,

	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x02,
	0x00, 0x00, 0x00, 0x03,

	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x02,
	0x00, 0x00, 0x00, 0x03,

	0x02, 0x00, 0x00, 0x00,
	0x11, 0x22, 0x33, 0x44,

	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x01,

	0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x00,
	0x00, 0x00, 0x00, 0x05,
	0x62, 0x61, 0x6e, 0x61, 0x6e,
}

func TestSizeof(t *testing.T) {
	size, err := Sizeof(&refStruct)
	if err != nil {
		t.Error(err)
	} else if size != 101 {
		t.Error("Bad sizeof result", size, "expected", 101)
	}
}

func TestEncode(t *testing.T) {
	buf := &bytes.Buffer{}
	err := EncodeWithOrder(buf, &refStruct, binary.BigEndian)
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(buf.Bytes(), refBytes) {
		t.Error("Bad encode result")
		t.Error("expected:", hex.EncodeToString(buf.Bytes()))
		t.Error("received:", hex.EncodeToString(refBytes))
	}
}

func TestDecode(t *testing.T) {
	buf := bytes.NewBuffer(refBytes)
	ret := testStruct{}
	err := DecodeWithOrder(buf, &ret, binary.BigEndian)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(ret, refStruct) {
		t.Error("Bad decode result")
		t.Error("expected:", refStruct)
		t.Error("received:", ret)
	}
}
