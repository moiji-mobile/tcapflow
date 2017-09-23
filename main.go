package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"strconv"
	"time"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"gopkg.in/alexcesaro/statsd.v2"
)

type DataHandler interface {
	HandleData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8)
}

type TCAPDialogueStart struct {
	StartTime	time.Time
	Ros		[]ROSInfo
	Otid		[]byte
}

type TCAPFlowDataHandler struct {
	Sessions	map[string]TCAPDialogueStart
	Scale		time.Duration
	Statsd		*statsd.Client
	ExpireDuration	time.Duration
}

func buildKey(gt SCCPAddress, tid []byte) string {
	return gt.Number + "-" + strconv.Itoa(int(gt.Ssn)) + "-" + hex.EncodeToString(tid)
}

func addState(t *TCAPFlowDataHandler, called_gt, calling_gt SCCPAddress, otid []byte, infos []ROSInfo) {
	key := buildKey(calling_gt, otid)
	elem := TCAPDialogueStart{
			StartTime: time.Now(),
			Ros: infos,
			Otid: otid}
	t.Sessions[key] = elem
	t.Statsd.Increment("tcapflow.newState")
}

func removeState(t *TCAPFlowDataHandler, called_gt, calling_gt SCCPAddress, dtid []byte, infos []ROSInfo) {
	key := buildKey(called_gt, dtid)
	val, ok := t.Sessions[key]
	now := time.Now()

	if ok {
		diff := now.Sub(val.StartTime)
		delete(t.Sessions, key)
		t.Statsd.Increment("tcapflow.delState")
		t.Statsd.Timing("tcapflow.latency", float64(diff / t.Scale))
	}

	// Expire older sessions. With seconds we run into problems...
	// maybe only run once every X runs..
	for key, value := range t.Sessions {
		diff := now.Sub(value.StartTime)
		if diff > t.ExpireDuration {
			t.Statsd.Increment("tcapflow.expiredState")
			delete(t.Sessions, key)
		}
	}
}

func procName(tag int) string {
	switch tag {
	case TCbeginApp:
		return "BEGIN"
	case TCendApp:
		return "END"
	case TCcontinueApp:
		return "CONTINUE"
	case TCabortApp:
		return "ABORT"
	default:
		return strconv.Itoa(tag)
	}
}

func (t *TCAPFlowDataHandler) HandleData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8) {
	tag, otid, dtid, _, comp, _ := decodeTCAP(data)
	infos, _ := decodeROS(comp.Bytes)

	switch tag {
	case TCbeginApp:
		fmt.Printf("BEGIN OTID(%v) %v->%v STATES(%v)", otid.Bytes, calling_gt.Number, called_gt.Number, len(t.Sessions))
		addState(t, called_gt, calling_gt, otid.Bytes, infos)
		fmt.Printf("\n")
	case TCabortApp:
		fmt.Printf("ABORT ")
		t.Statsd.Increment("tcapflow.abort")
		fallthrough
	case TCendApp, TCcontinueApp:
		fmt.Printf("%s DTID(%v) %v<-%v STATES(%v)", procName(tag), dtid.Bytes, called_gt.Number, calling_gt.Number, len(t.Sessions))
		removeState(t, called_gt, calling_gt, dtid.Bytes, infos)
		fmt.Printf("\n")
	}

}

func handleM2UA(handler DataHandler, data *layers.SCTPData) {
	fmt.Printf("M2UA not implemented\n")
}

func handleSUA(handler DataHandler, data *layers.SCTPData) {
	fmt.Printf("SUA not implemented\n")
}

func handleSCTPData(handler DataHandler, data *layers.SCTPData) {
	switch (data.PayloadProtocol) {
	case layers.SCTPPayloadM2UA:
		handleM2UA(handler, data)
	case layers.SCTPPayloadM3UA:
		handleM3UA(handler, data.Payload)
	case layers.SCTPPayloadM2PA:
		handleM2PA(handler, data.Payload)
	case layers.SCTPPayloadSUA:
		handleSUA(handler, data)
	}
}

func handlePacket(handler DataHandler, packet gopacket.Packet) {
	for _, p := range packet.Layers() {
		if data, err := p.(*layers.SCTPData); err {
			handleSCTPData(handler, data)
		}
	}
}

func main() {
	var err error
	flowHandler := TCAPFlowDataHandler{}
	flowHandler.Sessions = make(map[string]TCAPDialogueStart)
	flowHandler.Scale = time.Millisecond

	// flags...
	pcapFile := flag.String("pcap-file", "", "Filename for PCAP")
	pcapDevice := flag.String("pcap-device", "any", "Device to sniff")
	pcapFilter := flag.String("pcap-filter", "sctp", "Filter for live sniffing")
	expireDuration := flag.Duration("expire-state", 10 * time.Second, "Remove state")
	statsdPrefix := flag.String("statsd-prefix", "", "Prefix for statsd messages")
	flag.Parse()

	flowHandler.ExpireDuration = *expireDuration
	flowHandler.Statsd, err = statsd.New(statsd.Prefix(*statsdPrefix))
	if err != nil {
		fmt.Printf("ERROR: Failed to create statsd client\n")
		return
	}
	defer flowHandler.Statsd.Close()

	// Open file or live...
	var handle *pcap.Handle
	if len(*pcapFile) > 0 {
		handle, err = pcap.OpenOffline(*pcapFile)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}
	} else {
		handle, err = pcap.OpenLive(*pcapDevice, 0, true,  pcap.BlockForever)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}
		err = handle.SetBPFFilter(*pcapFilter)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}
	}

	// Main loop..
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for {
		packet, err := packetSource.NextPacket()
		if err == io.EOF {
			break
		} else if err == nil {
			handlePacket(&flowHandler, packet)
			flowHandler.Statsd.Flush()
		}
	}

	// Debugging in case of ran with a PCAP file
	for _, val := range flowHandler.Sessions {
		fmt.Printf("LEFT OTID(%v) OTID_HEX(%v)\n", val.Otid, hex.EncodeToString(val.Otid))
	}
}
