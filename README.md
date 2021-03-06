wire [![GoDoc](https://godoc.org/github.com/dajoh/wire?status.png)](https://godoc.org/github.com/dajoh/wire) [![Build Status](https://travis-ci.org/dajoh/wire.svg?branch=master)](https://travis-ci.org/dajoh/wire) [![Coverage Status](https://coveralls.io/repos/dajoh/wire/badge.svg?branch=master&service=github)](https://coveralls.io/github/dajoh/wire?branch=master)
----

Wire provides an easy and flexible way to serialize and deserialize
Go structures to binary.
It has support for arrays, variable length slices and strings, embedded
structures, and even slices and arrays of embedded structures.

Wire serializes in little endian by default, but this can be overridden with
the use of struct field tags or by using the WithOrder functions.

The following tags are supported:
* `big` tells wire to (de)serialize the value in big endian
* `little` tells wire to (de)serialize the value in little endian
* `nullterm` tells wire to (de)serialize the string with a null terminator
* `sizeof=$` tells wire that this field contains the length of another field

```go
type Example struct {
  Cmd         uint8
  UsernameLen uint16 `wire:"sizeof=Username,big"`
  Username    string
  Password    string `wire:"nullterm"`
}

// Note that the value passed in must be a pointer as UsernameLen is modified!
wire.Encode(writer, &Example{Cmd: 1, Username: "dajoh", Password: "x"})
```
