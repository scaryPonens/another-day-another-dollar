package docs

import "testing"

func TestSwaggerInfoRegistered(t *testing.T) {
	if SwaggerInfo == nil {
		t.Fatal("swagger info not initialized")
	}
	if SwaggerInfo.Title == "" {
		t.Fatal("swagger info missing title")
	}
}
