package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestCheckHealth(t *testing.T) {
	ds := &Datasource{}
	
	// Case 1: Missing API Token
	settings := backend.DataSourceInstanceSettings{
		JSONData:                []byte(`{"url":"https://test.com", "username":"user"}`),
		DecryptedSecureJSONData: map[string]string{},
	}
	
	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &settings,
		},
	}

	res, _ := ds.CheckHealth(context.Background(), req)
	
	if res.Status != backend.HealthStatusError {
		t.Errorf("expected HealthStatusError for missing token, got %v", res.Status)
	}
	
	if res.Message != "API Token is missing" {
		t.Errorf("expected 'API Token is missing', got '%s'", res.Message)
	}
}
