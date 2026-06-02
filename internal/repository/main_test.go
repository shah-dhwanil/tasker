package repository_test

import (
	"os"
	"testing"

	"github.com/shah-dhwanil/tasker/internal/errors"
	pkgTesting "github.com/shah-dhwanil/tasker/internal/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertAppErrorType(t *testing.T, err error, expectedType errors.ErrorType) {
	t.Helper()
	require.Error(t, err)
	var appErr *errors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, expectedType, appErr.Type)
}

func TestMain(m *testing.M) {
    cleanup := pkgTesting.Setup()
    code := m.Run()
    cleanup()
    os.Exit(code)
}