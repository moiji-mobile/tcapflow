package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/moiji-mobile/tcapflow"
	"github.com/moiji-mobile/tcapflow/rpc"


	"gopkg.in/alexcesaro/statsd.v2"
)

type TCAPDialogueStart struct {
	CaptTime	time.Time
	AddedTime	time.Time
	Ros		[]*rpc.ROSInfo
	Otid		[]byte
}

// The path from one node to the server might be more quick than
// the other. We will have to queue some responses for that.
type TCAPEarlyStateInfo struct {
	State		rpc.StateInfo
	AddedTime	time.Time	// When was the state locally added?
	CaptTime	time.Time
}

// For dialogues we stopped to track but might add more messages
type TCAPOld struct {
	EndedTime	time.Time
}

type TCAPFlowServer struct {
	Sessions	map[string]TCAPDialogueStart // TC-begin one waiting for a pick-up
	EarlyPending	map[string]TCAPEarlyStateInfo
	Old		map[string]TCAPOld // TC-end or second TC-continue

	Statsd		*statsd.Client

	Scale		time.Duration
	ExpireSessionDuration time.Duration
	ExpirePendingDuration time.Duration
	ExpireEndedDuration time.Duration
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
			CaptTime: capt,
			Ros: infos,
			Otid: otid}
	t.Sessions[key] = elem
	t.Statsd.Increment("tcapflow-server.newState")

	// Check if a pending end can be applied now
	early, ok := t.EarlyPending[key]
	if ok {
		state := early.State
		time, _ := ptypes.Timestamp(state.Time)
		doRemoveState(t, time, *state.Called, state.Tcap.Dtid, state.Tcap.Tag)
	}

	removeOldSessions(t)
}

func doRemoveState(t *TCAPFlowServer, capt time.Time, called_gt rpc.SCCPAddress, dtid []byte, tag int32)  bool {
	key := buildKey(called_gt, dtid)
	val, ok := t.Sessions[key]

	if !ok {
		return false
	}

	diff := capt.Sub(val.CaptTime)
	delete(t.Sessions, key)
	t.Statsd.Increment("tcapflow-server.delState")
	t.Statsd.Timing("tcapflow-server.latency", float64(diff / t.Scale))

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
		t.Old[key] = TCAPOld{EndedTime: time.Now()}
	}

	return true
}

func removeState(t *TCAPFlowServer, capt time.Time, state rpc.StateInfo) {
	called := *state.Called
	dtid := state.Tcap.Dtid

	// Is the state removed?
	if !doRemoveState(t, capt, called, dtid, state.Tcap.Tag) {
		// Check if it is already pending?
		key := buildKey(called, dtid)
		_, ok := t.EarlyPending[key]
		if !ok {
			// Check if it is known removed
			_, ok = t.Old[key]
			if !ok {
				// Let's remember it...
				t.EarlyPending[key] = TCAPEarlyStateInfo{
					State: state,
					AddedTime: time.Now(),
					CaptTime: capt,
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

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 6666))
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}

	flowServer := TCAPFlowServer{}
	flowServer.ExpireSessionDuration = 10 * time.Second
	flowServer.ExpirePendingDuration = 2 * time.Second
	flowServer.ExpireEndedDuration = 10 * time.Second

	grpcServer := grpc.NewServer()
	rpc.RegisterTCAPFlowServer(grpcServer, &flowServer)
	grpcServer.Serve(lis)
}