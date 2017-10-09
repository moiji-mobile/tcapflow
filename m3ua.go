package tcapflow

import (
	"encoding/binary"
	"bytes"
	"fmt"
)

type M3UA struct {
	Version		uint8
	Reserved	uint8
	MessageClass	uint8
	MessageType	uint8
	Length		uint32
}

type M3UAHeader struct {
	Tag		uint16
	Length		uint16
}

func HandleM3UA(handler DataHandler, data []uint8) {
	m3ua := M3UA{}
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &m3ua)
	if err != nil {
		fmt.Printf("Failed M3UA: %v\n", err)
		return
	}

	if m3ua.MessageClass != 11 && m3ua.MessageType != 1 {
		return
	}

	for buf.Len() >= 4 {
		hdr := M3UAHeader{}
		err = binary.Read(buf, binary.BigEndian, &hdr)
		if err != nil {
			break
		}

		payload := make([]byte, hdr.Length - 4)
		_, err = buf.Read(payload)
		if err != nil {
			break
		}
		if hdr.Tag == 528 {
			handleSCCP(handler, payload[12:])
		}
		if hdr.Length % 4 > 0 {
			padding := int(4 - (hdr.Length % 4))
			for i := 0; i < padding; i++ {
				_, err = buf.ReadByte()
				if err != nil {
					break
				}
			}
		}
	}
}

