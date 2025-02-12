package messages

const SetTraceMethod = "$/setTrace"

type SetTraceParams struct {
	Value TraceValueKind `json:"value"`
}

type TraceValueKind string

const (
	TraceValueOff      TraceValueKind = "off"
	TraceValueMessages TraceValueKind = "messages"
	TraceValueVerbose  TraceValueKind = "verbose"
)
