package report

import (
	"encoding/json"

	"github.com/oaslananka/cifence/internal/githubactions"
)

func JSON(report githubactions.Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
