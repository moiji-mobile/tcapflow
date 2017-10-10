package tcapflow

import (
	"github.com/google/gopacket"
)

type DataHandler interface {
	OnData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8, packet gopacket.Packet)
	AfterOnePacket()
	ParseError(data []uint8, recovered interface{})
}

