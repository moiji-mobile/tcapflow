package tcapflow

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func HandleSUA(handler DataHandler, data *layers.SCTPData, packet gopacket.Packet) {
	fmt.Printf("SUA not implemented\n")
}
