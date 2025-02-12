package messages

const LogMessageMethod = "window/logMessage"

type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}
