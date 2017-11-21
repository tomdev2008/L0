package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/bocheninc/L0/components/utils"
)

const (
	stringType = iota
)

func DoContractStateData(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	buf := bytes.NewBuffer(src)
	tp, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	switch tp {
	case stringType:
		len, err := utils.ReadVarInt(buf)
		if err != nil {
			return nil, err
		}
		data := make([]byte, len)
		buf.Read(data)
		return data, nil
	default:
		return nil, errors.New("not support states")
	}
}

func ConcrateStateJson(v interface{}) (*bytes.Buffer, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return &bytes.Buffer{}, err
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(stringType)
	lenByte := utils.VarInt(uint64(len(data)))
	buf.Write(lenByte)
	buf.Write(data)

	return buf, nil
}
