package messages

const ExecuteCommandMethod = "workspace/executeCommand"

type ExecuteCommandParams struct {
	WorkDoneProgressParams

	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}
