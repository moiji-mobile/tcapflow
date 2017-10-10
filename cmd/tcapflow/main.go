package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"
	"time"
	"gopkg.in/alexcesaro/statsd.v2"
	"github.com/google/gopacket"

	. "github.com/moiji-mobile/tcapflow"
)

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

func (t *TCAPFlowDataHandler) OnData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8, packet gopacket.Packet) {
	tag, otid, dtid, _, comp, _ := DecodeTCAP(data)
	infos, _ := DecodeROS(comp.Bytes)

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
		fmt.Printf("%s DTID(%v) %v<-%v STATES(%v)", TCprocName(tag), dtid.Bytes, called_gt.Number, calling_gt.Number, len(t.Sessions))
		removeState(t, called_gt, calling_gt, dtid.Bytes, infos)
		fmt.Printf("\n")
	}

}

func (t *TCAPFlowDataHandler) ParseError(data []uint8, r interface{}) {
	fmt.Printf("ParseError: SCTP(%v) %v\n", hex.EncodeToString(data), r)
	t.Statsd.Increment("tcapflow.parseError")
}

func (t *TCAPFlowDataHandler) AfterOnePacket() {
	t.Statsd.Flush()
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

	RunLoop(*pcapFile, *pcapDevice, *pcapFilter, &flowHandler)
}
