package httpapi

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIContractIsValid(t *testing.T) {
	document := loadOpenAPI(t)
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}
}

func TestOpenAPIRoutesMatchRouter(t *testing.T) {
	document := loadOpenAPI(t)
	want := make(map[string]struct{})
	for path, item := range document.Paths.Map() {
		for method := range item.Operations() {
			want[method+" /api/v1"+path] = struct{}{}
		}
	}

	got := make(map[string]struct{})
	for _, route := range NewRouter(RouterConfig{}).Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/") {
			got[route.Method+" "+normalizeGinPath(route.Path)] = struct{}{}
		}
	}

	if missing, extra := setDifference(want, got), setDifference(got, want); len(missing) != 0 || len(extra) != 0 {
		t.Fatalf("OpenAPI/router mismatch\nmissing from router: %v\nmissing from OpenAPI: %v", missing, extra)
	}
}

func TestOpenAPIOperationsHaveTypedJSONBodies(t *testing.T) {
	document := loadOpenAPI(t)
	for path, item := range document.Paths.Map() {
		for method, operation := range item.Operations() {
			t.Run(method+" "+path, func(t *testing.T) {
				if operation.Responses == nil || operation.Responses.Default() == nil {
					t.Fatal("operation must define the standard default error response")
				}
				for status, response := range operation.Responses.Map() {
					if !strings.HasPrefix(status, "2") || status == "204" {
						continue
					}
					if response.Value == nil || response.Value.Content.Get("application/json") == nil ||
						response.Value.Content.Get("application/json").Schema == nil {
						t.Fatalf("success response %s must have an application/json schema", status)
					}
				}
				if operation.RequestBody != nil {
					body := operation.RequestBody.Value
					if body == nil || body.Content.Get("application/json") == nil || body.Content.Get("application/json").Schema == nil {
						t.Fatal("request body must have an application/json schema")
					}
				}
			})
		}
	}
}

func loadOpenAPI(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(openAPISpec)
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	return document
}

func normalizeGinPath(path string) string {
	parts := strings.Split(path, "/")
	for index, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[index] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func setDifference(left, right map[string]struct{}) []string {
	result := make([]string, 0)
	for value := range left {
		if _, ok := right[value]; !ok {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
