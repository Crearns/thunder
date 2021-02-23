package protocol

import (
	"bytes"
	"encoding/binary"
)

const (
	headerFixedLength = 21
)

var (
	THUNDER *ThunderSerializer
)

func init() {
	THUNDER = &ThunderSerializer{}
}

type ThunderSerializer struct {
}

func (t *ThunderSerializer) Marshal(p *Packet) ([]byte, error) {
	extBytes, err := t.encodeMaps(p.ExtFields)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, headerFixedLength+len(p.Remark)+len(extBytes)))
	buf.Reset()

	// request code, length is 2 bytes
	err = binary.Write(buf, binary.BigEndian, int16(p.Code))
	if err != nil {
		return nil, err
	}

	// language flag, length is 1 byte
	err = binary.Write(buf, binary.BigEndian, Golang)
	if err != nil {
		return nil, err
	}

	// version flag, length is 2 bytes
	err = binary.Write(buf, binary.BigEndian, int16(p.Version))
	if err != nil {
		return nil, err
	}

	// opaque flag, opaque is request identifier, length is 4 bytes
	err = binary.Write(buf, binary.BigEndian, p.PacketId)
	if err != nil {
		return nil, err
	}

	// request flag, length is 4 bytes
	err = binary.Write(buf, binary.BigEndian, p.Flag)
	if err != nil {
		return nil, err
	}

	// remark length flag, length is 4 bytes
	err = binary.Write(buf, binary.BigEndian, int32(len(p.Remark)))
	if err != nil {
		return nil, err
	}

	// write remark, len(command.Remark) bytes
	if len(p.Remark) > 0 {
		err = binary.Write(buf, binary.BigEndian, []byte(p.Remark))
		if err != nil {
			return nil, err
		}
	}

	err = binary.Write(buf, binary.BigEndian, int32(len(extBytes)))
	if err != nil {
		return nil, err
	}

	if len(extBytes) > 0 {
		err = binary.Write(buf, binary.BigEndian, extBytes)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (t *ThunderSerializer) UnMarshal(data []byte) (*Packet, error) {
	var err error
	packet := &Packet{}
	buf := bytes.NewBuffer(data)
	var code int16
	err = binary.Read(buf, binary.BigEndian, &code)
	if err != nil {
		return nil, err
	}
	packet.Code = code

	var (
		languageCode byte
		remarkLen    int32
		extFieldsLen int32
	)
	err = binary.Read(buf, binary.BigEndian, &languageCode)
	if err != nil {
		return nil, err
	}
	packet.Language = LanguageCode(languageCode)

	var version int16
	err = binary.Read(buf, binary.BigEndian, &version)
	if err != nil {
		return nil, err
	}
	packet.Version = version

	// int opaque
	err = binary.Read(buf, binary.BigEndian, &packet.PacketId)
	if err != nil {
		return nil, err
	}

	// int flag
	err = binary.Read(buf, binary.BigEndian, &packet.Flag)
	if err != nil {
		return nil, err
	}

	// String remark
	err = binary.Read(buf, binary.BigEndian, &remarkLen)
	if err != nil {
		return nil, err
	}

	if remarkLen > 0 {
		var remarkData = make([]byte, remarkLen)
		err = binary.Read(buf, binary.BigEndian, &remarkData)
		if err != nil {
			return nil, err
		}
		packet.Remark = string(remarkData)
	}

	err = binary.Read(buf, binary.BigEndian, &extFieldsLen)
	if err != nil {
		return nil, err
	}

	if extFieldsLen > 0 {
		extFieldsData := make([]byte, extFieldsLen)
		err = binary.Read(buf, binary.BigEndian, &extFieldsData)
		if err != nil {
			return nil, err
		}

		packet.ExtFields = make(map[string]string)
		buf := bytes.NewBuffer(extFieldsData)
		var (
			kLen int16
			vLen int32
		)
		for buf.Len() > 0 {
			err = binary.Read(buf, binary.BigEndian, &kLen)
			if err != nil {
				return nil, err
			}

			key, err := getExtFieldsData(buf, int32(kLen))
			if err != nil {
				return nil, err
			}

			err = binary.Read(buf, binary.BigEndian, &vLen)
			if err != nil {
				return nil, err
			}

			value, err := getExtFieldsData(buf, vLen)
			if err != nil {
				return nil, err
			}
			packet.ExtFields[key] = value
		}
	}

	return packet, nil
}

func (t *ThunderSerializer) encodeMaps(maps map[string]string) ([]byte, error) {
	if maps == nil || len(maps) == 0 {
		return []byte{}, nil
	}
	extFieldsBuf := bytes.NewBuffer([]byte{})
	var err error
	for key, value := range maps {
		err = binary.Write(extFieldsBuf, binary.BigEndian, int16(len(key)))
		if err != nil {
			return nil, err
		}
		err = binary.Write(extFieldsBuf, binary.BigEndian, []byte(key))
		if err != nil {
			return nil, err
		}

		err = binary.Write(extFieldsBuf, binary.BigEndian, int32(len(value)))
		if err != nil {
			return nil, err
		}
		err = binary.Write(extFieldsBuf, binary.BigEndian, []byte(value))
		if err != nil {
			return nil, err
		}
	}
	return extFieldsBuf.Bytes(), nil
}

func getExtFieldsData(buff *bytes.Buffer, length int32) (string, error) {
	var data = make([]byte, length)
	err := binary.Read(buff, binary.BigEndian, &data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
