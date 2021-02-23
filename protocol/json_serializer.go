package protocol

import (
	"github.com/json-iterator/go"
)

var (
	JSON *JSONSerializer
)

func init() {
	JSON = &JSONSerializer{
		API: jsoniter.Config{
			EscapeHTML: false,
		}.Froze(),
	}
}

type JSONSerializer struct {
	jsoniter.API
}

func (j *JSONSerializer) Marshal(p *Packet) ([]byte, error) {
	return j.API.Marshal(p)
}

func (j *JSONSerializer) UnMarshal(bs []byte) (*Packet, error) {
	p := new(Packet)
	return p, j.API.Unmarshal(bs, p)
}
