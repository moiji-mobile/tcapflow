package tcapflow

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func HandleM2UA(handler DataHandler, data *layers.SCTPData, packet gopacket.Packet) {
	fmt.Printf("M2UA not implemented\n")
}

