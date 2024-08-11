package object

import (
	"encoding/binary"
	"fmt"
	"os"

	"monkey/code"
	"monkey/serializer"
)

func (c *CompiledFunction) Serialize() []byte {
	serializedFn := []byte{byte(serializer.COMPILEDFN)}

	instructionsLenBuf := make([]byte, 8)
	binary.PutVarint(instructionsLenBuf, int64(len([]byte(c.Instructions))))
	serializedFn = append(serializedFn, instructionsLenBuf...)
	serializedFn = append(serializedFn, c.Instructions...)

	numLocalsBuf := make([]byte, 8)
	binary.PutVarint(numLocalsBuf, int64(c.NumLocals))
	serializedFn = append(serializedFn, numLocalsBuf...)

	numParametersBuf := make([]byte, 8)
	binary.PutVarint(numParametersBuf, int64(c.NumParameters))
	serializedFn = append(serializedFn, numParametersBuf...)

	return serializedFn
}

func (c *CompiledFunction) Deserialize(bs []byte) int {
	instructionsLen, n := binary.Varint(bs[1:])
	if n < 0 || n > 8 || int(instructionsLen) > len(bs) - 25 {
		fmt.Fprintf(os.Stderr, "couldn't read instructions length, got %d bytes: length=%d bytes=%v", n, instructionsLen, bs[1:9])
		return -1 + n
	}

	numLocals, n := binary.Varint(bs[9 + int(instructionsLen):])
	if n < 0 || n > 8 {
		fmt.Fprintf(os.Stderr, "couldn't read numLocals=%d, got %d bytes", numLocals, n)
		return -9 - int(instructionsLen) + n
	}

	numParameters, n := binary.Varint(bs[17 + int(instructionsLen):])
	if n < 0 || n > 8 {
		fmt.Fprintf(os.Stderr, "couldn't read numParameters=%d, got %d bytes", numParameters, n)
		return -17 - int(instructionsLen) + n
	}

	c.Instructions = code.Instructions(bs[9 : 9 + int(instructionsLen)])
	c.NumLocals = int(numLocals)
	c.NumParameters = int(numParameters)

	return 25 + int(instructionsLen)
}

func (s *String) Serialize() []byte {
	serializedString := []byte{byte(serializer.STRING)}

	lenBuffer := make([]byte, 8)
	binary.PutVarint(lenBuffer, int64(len(s.Value)))
	serializedString = append(serializedString, lenBuffer...)

	serializedString = append(serializedString, []byte(s.Value)...)

	return serializedString
}

func (s *String) Deserialize(bs []byte) int {
	length, n := binary.Varint(bs[1:9])
	if n < 0 || n > 8 || int(length) < len(bs) - 9 {
		return -1 - n
	}
	s.Value = string(bs[9 : 9 + int(length)])
	return 9 + int(length)
}

func (i *Integer) Serialize() []byte {
	bs := make([]byte, 8)
	binary.PutVarint(bs, i.Value)
	return append([]byte{byte(serializer.INTEGER)}, bs...)
}

func (i *Integer) Deserialize(bs []byte) int {
	if len(bs) < 9 {
		return -1
	}
	val, n := binary.Varint(bs[1:])
	if n < 0 || n > 8 {
		return -1 + n
	}
	i.Value = val
	return 9
}
