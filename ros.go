package tcapflow

import (
	"encoding/asn1"
)

const (
	ROSInvoke = 1
	ROSResult = 2
)

type ROSInfo struct {
	Type     int
	InvokeId int
	OpCode   int
}

func decodeInvoke(data []byte) (info ROSInfo, err error) {
	info.Type = ROSInvoke

	data, err = asn1.Unmarshal(data, &info.InvokeId)
	if err != nil {
		return
	}
	data, err = asn1.Unmarshal(data, &info.OpCode)
	return
}

func decodeResult(data []byte) (info ROSInfo, err error) {
	info.Type = ROSResult
	info.OpCode = -1

	data, err = asn1.Unmarshal(data, &info.InvokeId)
	return
}

func DecodeROS(data []byte) (infos []ROSInfo, err error) {
	for len(data) > 0 {
		var tmp asn1.RawValue

		data, err = asn1.Unmarshal(data, &tmp)
		if err != nil {
			return
		}

		switch tmp.Tag {
		case ROSInvoke:
			info, err := decodeInvoke(tmp.Bytes)
			if err == nil {
				infos = append(infos, info)
			}
		case ROSResult:
			info, err := decodeResult(tmp.Bytes)
			if err == nil {
				infos = append(infos, info)
			}
		}
	}
	return
}
