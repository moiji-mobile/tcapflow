package tcapflow

import (
	"encoding/binary"
	"bytes"
	"fmt"
)

type MTPL3 struct {
	Service		uint8
	Routing		[4]uint8
}

func handleMTP(handler DataHandler, data []uint8) {
	mtpl3 := MTPL3{}
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &mtpl3)
	if err != nil {
		fmt.Printf("Failed MTP: %v\n", err)
		return
	}
	if (mtpl3.Service & 0x0f) == 0x03 {
		handleSCCP(handler, data[5:])
	}
}

