package messages

const CancelRequestMethod = "$/cancelRequest"

type CancelRequestParams struct {
	ID any `json:"id"`
}
