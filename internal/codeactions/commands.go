package codeactions

func GetCommandNames() []string {
	return codeActionCommandNames
}

const (
	commandApply       = "APPLY"
	commandDelete      = "DELETE"
	commandApplyDryRun = "APPLY_DRY_RUN"
)

var codeActionCommandNames = []string{
	commandApply,
	commandApplyDryRun,
	commandDelete,
}

var codeActionCommands = map[string]struct {
	Title          string
	FailedMessage  string
	SuccessMessage string
}{
	commandApply: {
		Title:          "Apply objects defined in this file",
		FailedMessage:  "Failed to apply objects",
		SuccessMessage: "Objects applied successfully",
	},
	commandApplyDryRun: {
		Title:          "Apply objects defined in this file (dry-run)",
		FailedMessage:  "Failed to apply objects (dry-run)",
		SuccessMessage: "Objects applied successfully (dry-run)",
	},
	commandDelete: {
		Title:          "Delete objects defined in this file",
		FailedMessage:  "Failed to delete objects",
		SuccessMessage: "Objects deleted successfully",
	},
}
