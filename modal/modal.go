package modal

import "encoding/json"

type FirePacket struct {
	Data    []byte `json:"data"`
	Counter int    `json:"counter"`
	Request int    `json:"request"`
}

func (fp FirePacket) GetBytePacket() (bytedata []byte, err error) {
	return json.Marshal(fp)
}
