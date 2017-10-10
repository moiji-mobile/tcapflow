package tcapflow

import (
	"encoding/binary"
	"bytes"
	"fmt"

	"github.com/google/gopacket"
)

type SCCPUDT struct {
	MessageType	uint8
	Options		uint8
	FirstMandatory	uint8
	SecondMandatory	uint8
	ThirdMandatory	uint8
}

type SCCPAddress struct {
	Ssn		uint8
	Ton		uint8
	Npi		uint8
	Number		string
}

func parseAddr(data []uint8) (addr SCCPAddress, err error) {
	if len(data) <  5 {
		err = fmt.Errorf("Not enough bytes for SSN and GT: %#v", data)
		return
	}

	addr.Ssn = data[1]
	addr.Ton = data[3] & 0xF0 >> 4
	oddEven := data[3] & 0x01 == 1
	addr.Npi = data[4]

	number := make([]byte, 0, 16)

	for i := 5; i < len(data); i++ {
		nibble := data[i] & 0x0F
		number = append(number, nibble + 48)
		nibble = data[i] & 0xF0 >> 4
		number = append(number, nibble + 48)
	}
	if oddEven {
		addr.Number = string(number[0:len(number)-1])
	} else {
		addr.Number = string(number)
	}
	return
}

func handleSCCP(handler DataHandler, data []uint8, packet gopacket.Packet) {
	sccp := SCCPUDT{}
	if data[0] != 0x09 {
		fmt.Printf("SCCP: Not unitdata\n")
		return
	}
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &sccp)
	if err != nil {
		fmt.Printf("Failed SCCP: %v\n", err)
		return
	}

	offset := 2 + sccp.FirstMandatory
	calledLen := data[offset]
	calledDat := data[offset + 1: offset + 1 + calledLen]
	calledAddr, err := parseAddr(calledDat)

	offset = 3 + sccp.SecondMandatory
	callingLen := data[offset]
	callingDat := data[offset + 1: offset + 1 + callingLen]
	callingAddr, err := parseAddr(callingDat)

	offset = 4 + sccp.ThirdMandatory
	payloadLen := data[offset]
	payloadDat := data[offset + 1: offset + 1 + payloadLen]

	handler.OnData(calledAddr, callingAddr, payloadDat, packet)
}

