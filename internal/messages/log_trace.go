package messages

const LogTraceMethod = "$/logTrace"

type LogTraceParams struct {
	Message string `json:"message"`
	Verbose string `json:"verbose,omitempty"`
}
