package messages

const ShowMessageMethod = "window/showMessage"

type ShowMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

type MessageType int

const (
	MessageTypeError MessageType = iota + 1
	MessageTypeWarning
	MessageTypeInfo
	MessageTypeLog
	MessageTypeDebug
)
