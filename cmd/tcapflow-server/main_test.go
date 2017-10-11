package main

import (
	"testing"

	"golang.org/x/net/context"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/moiji-mobile/tcapflow"
	"github.com/moiji-mobile/tcapflow/rpc"
)


func buildTcBegin() rpc.StateInfo {
	t := &timestamp.Timestamp {0, 0}
	return rpc.StateInfo{
			Time: t,
			Calling: &rpc.SCCPAddress{
					Ssn: 1,
					Ton: 23,
					Npi: 23,
					Number: "vlr"},
			Called: &rpc.SCCPAddress{
					Ssn: 2,
					Ton: 23,
					Npi: 23,
					Number: "hlr"},
			Tcap: &rpc.TCAPInfo{
					Otid: []byte{1, 2, 3, 4},
					Tag: tcapflow.TCbeginApp,
			}}
}

func buildTcEnd() rpc.StateInfo {
	t := &timestamp.Timestamp {1, 0}
	return rpc.StateInfo{
			Time: t,
			Calling: &rpc.SCCPAddress{
					Ssn: 2,
					Ton: 23,
					Npi: 23,
					Number: "hlr"},
			Called: &rpc.SCCPAddress{
					Ssn: 1,
					Ton: 23,
					Npi: 23,
					Number: "vlr"},
			Tcap: &rpc.TCAPInfo{
					Dtid: []byte{1, 2, 3, 4},
					Tag: tcapflow.TCendApp,
			}}
}

func buildTcContinue() rpc.StateInfo {
	t := &timestamp.Timestamp {1, 0}
	return rpc.StateInfo{
			Time: t,
			Calling: &rpc.SCCPAddress{
					Ssn: 2,
					Ton: 23,
					Npi: 23,
					Number: "hlr"},
			Called: &rpc.SCCPAddress{
					Ssn: 1,
					Ton: 23,
					Npi: 23,
					Number: "vlr"},
			Tcap: &rpc.TCAPInfo{
					Dtid: []byte{1, 2, 3, 4},
					Otid: []byte{4, 3, 2, 1},
					Tag: tcapflow.TCcontinueApp,
			}}
}

func TestTcBeginTcEnd(t *testing.T) {
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	e := buildTcEnd()

	// TC-begin first
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 1 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-end to finish it
	s.AddState(context.Background(), &e)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}
}

func TestTcBeginTcContinue(t *testing.T) {
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	c := buildTcContinue()

	// TC-begin first
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 1 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-continue to finish it
	s.AddState(context.Background(), &c)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 {
		t.Fatalf("Should have no data %v\n", len(s.EarlyPending))
	}
	if len(s.Old) != 1 {
		t.Fatalf("Should remember one old %v\n", len(s.Old))
	}
}

func TestTcBeginTcContinueTcEnd(t *testing.T) {
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	c := buildTcContinue()
	e := buildTcEnd()

	// TC-begin first
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 1 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-continue to finish it
	s.AddState(context.Background(), &c)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 {
		t.Fatalf("Should have no data %v\n", len(s.EarlyPending))
	}
	if len(s.Old) != 1 {
		t.Fatalf("Should remember one old %v\n", len(s.Old))
	}

	// TC-end now it ends..
	s.AddState(context.Background(), &e)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}
}

func TestTcBeginTcContinueTcEndTcEnd(t *testing.T) {
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	c := buildTcContinue()
	e := buildTcEnd()

	// TC-begin first
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 1 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-continue to finish it
	s.AddState(context.Background(), &c)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 {
		t.Fatalf("Should have no data %v\n", len(s.EarlyPending))
	}
	if len(s.Old) != 1 {
		t.Fatalf("Should remember one old %v\n", len(s.Old))
	}

	// TC-end now it ends..
	s.AddState(context.Background(), &e)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// Fake end.. should be coming from the other direction but good enough
	// to check the behavior of the code
	s.AddState(context.Background(), &e)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 1 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}
}

func TestTcBeginTcContinueTcContinue(t *testing.T) {
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	c := buildTcContinue()

	// TC-begin first
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 1 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-continue to finish it
	s.AddState(context.Background(), &c)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 1 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-continue should not add...
	s.AddState(context.Background(), &c)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 1 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}
}

func TestTcContinueTcBegin(t *testing.T) {
	// Order of arrival changed..
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	c := buildTcContinue()

	// TC-end arrived first
	s.AddState(context.Background(), &c)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 1 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-begin now
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 1 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}
}

func TestTcEndTcBegin(t *testing.T) {
	// Order of arrival changed..
	s := NewTCAPFlowServer()
	b := buildTcBegin()
	e := buildTcEnd()

	// TC-end arrived first
	s.AddState(context.Background(), &e)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 1 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

	// TC-begin now
	s.AddState(context.Background(), &b)
	if len(s.Sessions) != 0 {
		t.Fatalf("Should have one session %v\n", len(s.Sessions))
	}
	if len(s.EarlyPending) != 0 || len(s.Old) != 0 {
		t.Fatalf("Should have no data %v %v\n", len(s.EarlyPending), len(s.Old))
	}

}
