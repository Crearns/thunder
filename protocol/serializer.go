package protocol

type SerializeType byte

type Serializer interface {
	Marshal(p *Packet) ([]byte,error)
	UnMarshal(bs []byte) (*Packet, error)
}