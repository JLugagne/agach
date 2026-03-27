package queries_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestController() *controller.Controller {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetOutput(io.Discard)
	return controller.NewController(logger)
}

func newValidProjectID() domain.ProjectID {
	return domain.NewProjectID()
}

func newValidTaskID() domain.TaskID {
	return domain.NewTaskID()
}

func mustParseJSON(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}
