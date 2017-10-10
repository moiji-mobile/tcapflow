package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"gopkg.in/alexcesaro/statsd.v2"
	"google.golang.org/grpc"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/gopacket"
	. "github.com/moiji-mobile/tcapflow"
	"github.com/moiji-mobile/tcapflow/rpc"
)

type ClientFlowDataHandler struct {
	Statsd		*statsd.Client
	RpcClient	rpc.TCAPFlowClient
}

func SCCPAddressProto(addr SCCPAddress) *rpc.SCCPAddress {
	return &rpc.SCCPAddress{
		Ssn: uint32(addr.Ssn),
		Ton: uint32(addr.Ton),
		Npi: uint32(addr.Npi),
		Number: addr.Number,
	}
}

func ROSInfoProto(infos []ROSInfo) []*rpc.ROSInfo {
	rpcInfos := make([]*rpc.ROSInfo, 0, len(infos))
	for _, info := range infos {
		rpcInfos = append(rpcInfos, &rpc.ROSInfo {
					Type: int32(info.Type),
					InvokeId: int32(info.InvokeId),
					OpCode: int32(info.OpCode),
				})
	}
	return rpcInfos
}

func (t *ClientFlowDataHandler) OnData(called_gt SCCPAddress, calling_gt SCCPAddress, data []uint8, packet gopacket.Packet) {
	tag, otid, dtid, _, comp, _ := DecodeTCAP(data)
	infos, _ := DecodeROS(comp.Bytes)

	rpcTime, _ := ptypes.TimestampProto(packet.Metadata().Timestamp)
	rpc := &rpc.StateInfo{
			Time: rpcTime,
			Calling: SCCPAddressProto(calling_gt),
			Called: SCCPAddressProto(called_gt),
			Tcap: &rpc.TCAPInfo{
				Otid: otid.Bytes,
				Dtid: dtid.Bytes,
				Tag: int32(tag), },
			Ros: ROSInfoProto(infos),
		}

	_, err := t.RpcClient.AddState(context.Background(), rpc)
	if err != nil {
		fmt.Printf("RPC error: (%v)\n", err)
		t.Statsd.Increment("tcapflow-client.rpcError")
	}
}

func (t *ClientFlowDataHandler) ParseError(data []uint8, r interface{}) {
	fmt.Printf("ParseError: SCTP(%v) %v\n", hex.EncodeToString(data), r)
	t.Statsd.Increment("tcapflow-client.parseError")
}

func (t *ClientFlowDataHandler) AfterOnePacket() {
	t.Statsd.Flush()
}

func main() {
	var err error
	flowHandler := ClientFlowDataHandler{}

	// flags...
	pcapFile := flag.String("pcap-file", "", "Filename for PCAP")
	pcapDevice := flag.String("pcap-device", "any", "Device to sniff")
	pcapFilter := flag.String("pcap-filter", "sctp", "Filter for live sniffing")
	statsdPrefix := flag.String("statsd-prefix", "", "Prefix for statsd messages")
	serverAddr := flag.String("remote-address", "localhost:5345", "Hostname:port for RPC")
	flag.Parse()

	flowHandler.Statsd, err = statsd.New(statsd.Prefix(*statsdPrefix))
	if err != nil {
		fmt.Printf("ERROR: Failed to create statsd client\n")
		return
	}
	defer flowHandler.Statsd.Close()

	rpcConn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("ERROR: Failed to open RPC connection\n")
		return
	}
	defer rpcConn.Close()
	flowHandler.RpcClient = rpc.NewTCAPFlowClient(rpcConn)
	RunLoop(*pcapFile, *pcapDevice, *pcapFilter, &flowHandler)
}
