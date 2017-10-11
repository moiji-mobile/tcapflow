package tcapflow

import (
	"fmt"
	"io"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func reportParseError(handler DataHandler, data []uint8) {
	if r := recover(); r != nil {
		handler.ParseError(data, r)
	}

}

func handleSCTPData(handler DataHandler, data *layers.SCTPData, packet gopacket.Packet) {
	defer reportParseError(handler, data.Payload)

	switch (data.PayloadProtocol) {
	case layers.SCTPPayloadM2UA:
		HandleM2UA(handler, data, packet)
	case layers.SCTPPayloadM3UA:
		HandleM3UA(handler, data.Payload, packet)
	case layers.SCTPPayloadM2PA:
		HandleM2PA(handler, data.Payload, packet)
	case layers.SCTPPayloadSUA:
		HandleSUA(handler, data, packet)
	}
}

func handlePacket(handler DataHandler, packet gopacket.Packet) {
	for _, p := range packet.Layers() {
		if data, err := p.(*layers.SCTPData); err {
			handleSCTPData(handler, data, packet)
		}
	}
}

func RunLoop(pcapFile string, pcapDevice string, pcapFilter string, handler DataHandler) {
	// Open file or live...
	var handle *pcap.Handle
	var err error
	if len(pcapFile) > 0 {
		handle, err = pcap.OpenOffline(pcapFile)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}
	} else {
		handle, err = pcap.OpenLive(pcapDevice, 0, true,  pcap.BlockForever)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}
		err = handle.SetBPFFilter(pcapFilter)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}
	}
	defer handle.Close()

	// Main loop..
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for {
		packet, err := packetSource.NextPacket()
		if err == io.EOF {
			break
		} else if err == nil {
			handlePacket(handler, packet)
			handler.AfterOnePacket()
		}
	}
}
