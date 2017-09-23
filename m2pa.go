package main

import (
	"encoding/binary"
	"bytes"
	"fmt"
)

type M2PA struct {
	Version		uint8
	Spare		uint8
	MessageClass	uint8
	MessageType	uint8
	Length		uint32
	Unused		uint8
	Bsn		[3]uint8
	Unused2		uint8
	Fsn		[3]uint8
	Priority	uint8
}

func handleM2PA(handler DataHandler, data []uint8) {
	m2pa := M2PA{}
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &m2pa)
	if err != nil {
		fmt.Printf("Failed M2PA: %v\n", err)
		return
	}
	if m2pa.MessageClass == 11 && m2pa.MessageType == 1 {
		handleMTP(handler, data[17:])
		return
	}
}

