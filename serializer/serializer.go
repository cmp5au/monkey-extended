package serializer

type Serializable interface {
	Serialize() []byte
	Deserialize([]byte) int
}

type ObjectSerialType byte

const (
	_ = iota
	INTEGER ObjectSerialType = iota
	STRING
	COMPILEDFN
	BYTECODE
)
