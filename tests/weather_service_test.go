package tests

import (
	"context"
	"testing"
	"time"

	"github.com/agriconnect-ai/internal/weather"
)

type mockWeatherRepo struct {
	cacheData map[string]map[string]interface{}
	cacheTime map[string]time.Time
}

func (m *mockWeatherRepo) GetCache(_ context.Context, district string) (map[string]interface{}, *time.Time, error) {
	if data, ok := m.cacheData[district]; ok {
		t := m.cacheTime[district]
		return data, &t, nil
	}
	return nil, nil, nil
}

func (m *mockWeatherRepo) SetCache(_ context.Context, district string, data map[string]interface{}) error {
	m.cacheData[district] = data
	m.cacheTime[district] = time.Now().UTC()
	return nil
}

func TestDistrictValidation(t *testing.T) {
	if !weather.IsValidDistrict("Bo") {
		t.Error("expected Bo to be valid")
	}
	if !weather.IsValidDistrict("bo") {
		t.Error("expected bo (lowercase) to be valid")
	}
	if weather.IsValidDistrict("InvalidDistrict") {
		t.Error("expected InvalidDistrict to be invalid")
	}
}

func TestSupportedDistricts(t *testing.T) {
	districts := weather.SupportedDistricts
	if len(districts) == 0 {
		t.Fatal("expected at least one supported district")
	}
	found := false
	for _, d := range districts {
		if d == "Bo" {
			found = true
		}
	}
	if !found {
		t.Error("expected Bo in supported districts")
	}
}

func TestGetDistrictCoordinates(t *testing.T) {
	coord, ok := weather.GetDistrict("Kenema")
	if !ok {
		t.Fatal("expected Kenema to be found")
	}
	if coord.Name != "Kenema" {
		t.Errorf("expected Kenema, got %s", coord.Name)
	}
	if coord.Latitude == 0 || coord.Longitude == 0 {
		t.Error("expected non-zero coordinates")
	}
}
