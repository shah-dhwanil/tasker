package repository_test

import (
	"os"
	"testing"
	pkgTesting "github.com/shah-dhwanil/tasker/internal/testing"
)


func TestMain(m *testing.M) {
    cleanup := pkgTesting.Setup()
    code := m.Run()
    cleanup()
    os.Exit(code)
}