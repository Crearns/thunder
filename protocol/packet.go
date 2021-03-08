package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync/atomic"
)

const (
	RPCOneWay = 2
	ResponseType = 1
)

var (
	packetIdGenerator int32
	codecType         byte
)

type (
	LanguageCode byte
)

type Packet struct {
	Code     int16             `json:"code"`
	Language LanguageCode      `json:"language"`
	Version  int16             `json:"version"`
	PacketId int32             `json:"packetId"`
	Flag     int32             `json:"flag"`
	Message  string            `json:"message"`
	ExtData  map[string]string `json:"extData"`
	Body     []byte            `json:"-"`
}

func NewPacket(code int16, body []byte, header ExtData) *Packet {
	p := &Packet{
		Code:     code,
		Language: Golang,
		Version:  0,
		PacketId: atomic.AddInt32(&packetIdGenerator, 1),
		Body:     body,
	}

	if header != nil {
		p.ExtData = header.EncodeToMap()
	}

	return p
}

type ExtData interface {
	EncodeToMap() map[string]string
}

const (
	Golang  = LanguageCode(0)
	Java    = LanguageCode(1)
	Cpp     = LanguageCode(2)
	Python  = LanguageCode(3)
	Unknown = LanguageCode(127)
)

func (lc LanguageCode) MarshalJSON() ([]byte, error) {
	return []byte(`"GO"`), nil
}

func (lc *LanguageCode) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case "GO", `"GO"`:
		*lc = Golang
	default:
		*lc = Unknown
	}
	return nil
}

func (lc LanguageCode) String() string {
	switch lc {
	case Golang:
		return "GO"
	default:
		return "unknown"
	}
}

const (
	Json    = byte(0)
	Thunder = byte(1)
)

func (p *Packet) IsResponseType() bool {
	return p.Flag&(ResponseType) == ResponseType
}

func (p *Packet) MarkResponseType() {
	p.Flag = p.Flag | ResponseType
}

func (p *Packet) IsOneway() bool {
	return p.Flag&(RPCOneWay) == RPCOneWay
}

func (p *Packet) MarkOneway() {
	p.Flag = p.Flag | RPCOneWay
}

func MarkProtocolType(source int32) []byte {
	result := make([]byte, 4)
	result[0] = codecType
	result[1] = byte((source >> 16) & 0xFF)
	result[2] = byte((source >> 8) & 0xFF)
	result[3] = byte(source & 0xFF)
	return result
}

func Encode(packet *Packet) ([]byte, error) {
	var (
		header []byte
		err    error
	)

	switch codecType {
	case Json:
		header, err = JSON.Marshal(packet)
	case Thunder:
		header, err = THUNDER.Marshal(packet)
	}

	if err != nil {
		return nil, err
	}

	frameSize := len(header) + len(packet.Body) + 4 + 1
	buf := bytes.NewBuffer(make([]byte, frameSize))
	buf.Reset()

	err = binary.Write(buf, binary.BigEndian, codecType)
	if err != nil {
		return nil, err
	}

	headerLength := int32(len(header))
	err = binary.Write(buf, binary.BigEndian, headerLength)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, header)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, packet.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Decode(data []byte) (*Packet, error) {
	buf := bytes.NewBuffer(data)
	length := int32(len(data))
	var headerLength int32
	var codecTypeByte byte
	err := binary.Read(buf, binary.BigEndian, &codecTypeByte)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buf, binary.BigEndian, &headerLength)
	if err != nil {
		return nil, err
	}

	headerData := make([]byte, headerLength)
	err = binary.Read(buf, binary.BigEndian, &headerData)
	if err != nil {
		return nil, err
	}

	var packet *Packet
	switch codecTypeByte{
	case Json:
		packet, err = JSON.UnMarshal(headerData)
	case Thunder:
		packet, err = THUNDER.UnMarshal(headerData)
	default:
		err = fmt.Errorf("unknown codec type: %d", codecTypeByte)
	}
	if err != nil {
		return nil, err
	}

	bodyLength := length - 4 - 1 - headerLength
	if bodyLength > 0 {
		bodyData := make([]byte, bodyLength)
		err = binary.Read(buf, binary.BigEndian, &bodyData)
		if err != nil {
			return nil, err
		}
		packet.Body = bodyData
	}
	return packet, nil
}
