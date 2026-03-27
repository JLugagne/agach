package commands_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

func assertJSONFailCode(t *testing.T, data []byte, expectedCode string) {
	t.Helper()
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "fail", result["status"], "expected status=fail in response body")
	if data, ok := result["data"].(map[string]interface{}); ok {
		assert.Equal(t, expectedCode, data["code"], "expected error code %q", expectedCode)
	} else {
		t.Errorf("expected data.code=%q but data field is missing or not an object", expectedCode)
	}
}
