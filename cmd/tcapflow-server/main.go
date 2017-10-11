package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/moiji-mobile/tcapflow"
	"github.com/moiji-mobile/tcapflow/rpc"

	"gopkg.in/alexcesaro/statsd.v2"
)

type TCAPDialogueStart struct {
	CaptTime  time.Time
	AddedTime time.Time
	Ros       []*rpc.ROSInfo
	Otid      []byte
}

// The path from one node to the server might be more quick than
// the other. We will have to queue some responses for that.
type TCAPEarlyStateInfo struct {
	State     rpc.StateInfo
	AddedTime time.Time // When was the state locally added?
	CaptTime  time.Time
}

// For dialogues we stopped to track but might add more messages
type TCAPOld struct {
	EndedTime time.Time
}

type TCAPFlowServer struct {
	Sessions     map[string]TCAPDialogueStart // TC-begin one waiting for a pick-up
	EarlyPending map[string]TCAPEarlyStateInfo
	Old          map[string]TCAPOld // TC-end or second TC-continue

	Statsd *statsd.Client

	Scale                 time.Duration
	ExpireSessionDuration time.Duration
	ExpirePendingDuration time.Duration
	ExpireEndedDuration   time.Duration
}

func buildKey(gt rpc.SCCPAddress, tid []byte) string {
	return gt.Number + "-" + strconv.Itoa(int(gt.Ssn)) + "-" + hex.EncodeToString(tid)
}

func removeOldSessions(t *TCAPFlowServer) {
	now := time.Now()

	// Expire older sessions
	for key, value := range t.Sessions {
		diff := now.Sub(value.AddedTime)
		if diff > t.ExpireSessionDuration {
			t.Statsd.Increment("tcapflow-server.expiredState")
			delete(t.Sessions, key)
		}
	}

	// Expire old pending messages
	for key, value := range t.EarlyPending {
		diff := now.Sub(value.AddedTime)
		if diff > t.ExpirePendingDuration {
			t.Statsd.Increment("tcapflow-server.expiredEarlyPending")
			delete(t.EarlyPending, key)
		}
	}

	// Expire dead messages
	for key, value := range t.Old {
		diff := now.Sub(value.EndedTime)
		if diff > t.ExpireEndedDuration {
			t.Statsd.Increment("tcapflow-server.removedOldState")
			delete(t.Old, key)
		}
	}
}

func addState(t *TCAPFlowServer, capt time.Time, calling rpc.SCCPAddress, otid []byte, infos []*rpc.ROSInfo) {
	// Add the state
	key := buildKey(calling, otid)
	elem := TCAPDialogueStart{
		AddedTime: time.Now(),
		CaptTime:  capt,
		Ros:       infos,
		Otid:      otid}
	t.Sessions[key] = elem
	t.Statsd.Increment("tcapflow-server.newState")

	delete(t.Old, key)

	// Check if a pending end can be applied now
	early, ok := t.EarlyPending[key]
	if ok {
		delete(t.EarlyPending, key)
		state := early.State
		time, _ := ptypes.Timestamp(state.Time)
		removeState(t, time, state)
	}

	removeOldSessions(t)
}

func doRemoveState(t *TCAPFlowServer, key string, capt time.Time, called_gt rpc.SCCPAddress, dtid []byte, tag int32) bool {
	val, ok := t.Sessions[key]

	if !ok {
		return false
	}

	diff := capt.Sub(val.CaptTime)
	delete(t.Sessions, key)
	t.Statsd.Increment("tcapflow-server.delState")
	t.Statsd.Timing("tcapflow-server.latency", float64(diff/t.Scale))

	// Special work needed?
	_, ok = t.EarlyPending[key]
	if ok {
		delete(t.EarlyPending, key)
	}

	switch tag {
	case tcapflow.TCbeginApp:
		// Should never happen?
	case tcapflow.TCabortApp, tcapflow.TCendApp:
		// We are done for good!
	case tcapflow.TCcontinueApp:
		// Remember that more is to come
		t.Old[key] = TCAPOld{EndedTime: time.Now()}
	}

	removeOldSessions(t)
	return true
}

func removeState(t *TCAPFlowServer, capt time.Time, state rpc.StateInfo) {
	called := *state.Called
	dtid := state.Tcap.Dtid
	key := buildKey(called, dtid)

	// Is the state removed?
	if !doRemoveState(t, key, capt, called, dtid, state.Tcap.Tag) {
		// Not removed but maybe is old and it is over now?
		// Besides the point of both sides sending a TC-end and
		// the second is pending again. But such is life.
		_, isOld := t.Old[key]
		if isOld {
			switch state.Tcap.Tag {
			case tcapflow.TCendApp, tcapflow.TCabortApp:
				delete(t.Old, key)
			}
		} else {
			// Check if it is already pending?
			_, ok := t.EarlyPending[key]
			if !ok {
				// Let's remember it...
				t.EarlyPending[key] = TCAPEarlyStateInfo{
					State:     state,
					AddedTime: time.Now(),
					CaptTime:  capt,
				}
			}
		}
	}
}

func (t *TCAPFlowServer) AddState(ctx context.Context, in *rpc.StateInfo) (*empty.Empty, error) {

	// Missing mandatory fields
	if in.Calling == nil || in.Called == nil || in.Tcap == nil || in.Time == nil {
		t.Statsd.Increment("tcapflow-server.rpcMissingFields")
		return nil, nil
	}

	time, _ := ptypes.Timestamp(in.Time)

	switch in.Tcap.Tag {
	case tcapflow.TCbeginApp:
		addState(t, time, *in.Calling, in.Tcap.Otid, in.Ros)
	case tcapflow.TCabortApp:
		t.Statsd.Increment("tcapflow-server.tcAbort")
		fallthrough
	case tcapflow.TCendApp, tcapflow.TCcontinueApp:
		removeState(t, time, *in)
	}

	return nil, nil
}

func NewTCAPFlowServer() TCAPFlowServer {
	flowServer := TCAPFlowServer{}
	flowServer.ExpireSessionDuration = 10 * time.Second
	flowServer.ExpirePendingDuration = 2 * time.Second
	flowServer.ExpireEndedDuration = 10 * time.Second

	flowServer.Sessions = make(map[string]TCAPDialogueStart)
	flowServer.EarlyPending = make(map[string]TCAPEarlyStateInfo)
	flowServer.Old = make(map[string]TCAPOld)
	flowServer.Statsd, _ = statsd.New()

	flowServer.Scale = 1

	return flowServer
}

func main() {
	flowServer := NewTCAPFlowServer()
	statsdPrefix := flag.String("statsd-prefix", "", "Prefix for statsd messages")
	serverAddr := flag.String("listen-address", "localhost:5345", "Hostname:port for RPC")
	expireSession := flag.Duration("expire-session", flowServer.ExpireSessionDuration, "Time to keep unconfirmed TCAP dialogues")
	expirePending := flag.Duration("expire-pending", flowServer.ExpirePendingDuration, "Time to buffer messages for out-of-order arrival")
	expireEnded := flag.Duration("expired-ended", flowServer.ExpireEndedDuration, "Time to keep information of ended TCAP dialogues")
	flag.Parse()

	flowServer.ExpireSessionDuration = *expireSession
	flowServer.ExpirePendingDuration = *expirePending
	flowServer.ExpireEndedDuration = *expireEnded

	lis, err := net.Listen("tcp", *serverAddr)
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}
	flowServer.Statsd, err = statsd.New(statsd.Prefix(*statsdPrefix))
	if err != nil {
		fmt.Printf("ERROR: Failed to create statsd client\n")
		return
	}
	defer flowServer.Statsd.Close()

	grpcServer := grpc.NewServer()
	rpc.RegisterTCAPFlowServer(grpcServer, &flowServer)
	grpcServer.Serve(lis)
}
