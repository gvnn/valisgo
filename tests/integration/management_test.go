//go:build integration

package integration_test

import (
	"context"
	"testing"

	"valisgo/tests/integration/client"
)

func ptr[T any](v T) *T {
	return &v
}

func TestManagementClientIntegration(t *testing.T) {
	// Connect to the running dev server on the /manage prefix
	c, _ := client.NewClientWithResponses("http://localhost:8080/manage")
	ctx := context.Background()

	t.Run("Create and List", func(t *testing.T) {
		// Create
		createBody := client.CreateRegistryJSONRequestBody{
			Name:   "integration-reg",
			Format: ptr(client.RegistryInputFormatGo),
		}
		res, _ := c.CreateRegistryWithResponse(ctx, createBody)

		if res.StatusCode() != 201 && res.StatusCode() != 409 {
			t.Fatalf("expected 201 or 409, got %d", res.StatusCode())
		}

		// List
		listRes, _ := c.ListRegistriesWithResponse(ctx)
		if listRes.StatusCode() != 200 {
			t.Fatalf("expected 200, got %d", listRes.StatusCode())
		}
		
		if len(*listRes.JSON200) == 0 {
			t.Fatalf("expected at least 1 registry")
		}
	})
}
