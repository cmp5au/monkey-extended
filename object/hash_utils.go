package object

import (
	"encoding/binary"
	"hash/fnv"
)

func (s *String) Hash() HashKey {
	hash := fnv.New64a()
	hash.Write([]byte(s.Value))
	return HashKey{Type: s.Type(), Value: hash.Sum64()}
}

func (i *Integer) Hash() HashKey {
	hash := fnv.New64a()
	intBuffer := make([]byte, 8)
	binary.PutVarint(intBuffer, i.Value)
	hash.Write(intBuffer)
	return HashKey{Type: i.Type(), Value: hash.Sum64()}
}

func (b *Boolean) Hash() HashKey {
	var value uint64
	if !b.Value {
		value = ^value
	}
	return HashKey{Type: b.Type(), Value: value}
}
