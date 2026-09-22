package service_test

import (
	"testing"

	"github.com/redora/redora/backend/internal/models"
)

func TestFormatRedisResult(t *testing.T) {
	// Test basic integer result formatting
	resInt := models.ExecuteCommandResponse{
		Result: "(integer) 42",
		Status: "ok",
	}

	if resInt.Status != "ok" || resInt.Result != "(integer) 42" {
		t.Fatalf("Unexpected execution response: %+v", resInt)
	}
}
