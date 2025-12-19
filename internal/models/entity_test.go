package models

import (
	"testing"
)

func TestEntity_IsPrimaryEntity(t *testing.T) {
	tests := []struct {
		name     string
		entity   Entity
		expected bool
	}{
		{
			name: "Primary entity with high confidence",
			entity: Entity{
				Type:       EntityTypeCountry,
				Confidence: 0.85,
			},
			expected: true,
		},
		{
			name: "Primary type but low confidence",
			entity: Entity{
				Type:       EntityTypeCountry,
				Confidence: 0.5,
			},
			expected: false,
		},
		{
			name: "Non-primary type with high confidence",
			entity: Entity{
				Type:       EntityTypeOther,
				Confidence: 0.9,
			},
			expected: false,
		},
		{
			name: "Person entity high confidence",
			entity: Entity{
				Type:       EntityTypePerson,
				Confidence: 0.95,
			},
			expected: true,
		},
		{
			name: "Military unit high confidence",
			entity: Entity{
				Type:       EntityTypeMilitaryUnit,
				Confidence: 0.8,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entity.IsPrimaryEntity(); got != tt.expected {
				t.Errorf("IsPrimaryEntity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEntity_GetDisplayIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		entity   Entity
		expected string
	}{
		{
			name: "Normalized name present",
			entity: Entity{
				Name:           "U.S.A.",
				NormalizedName: "United States",
			},
			expected: "United States",
		},
		{
			name: "Only name present",
			entity: Entity{
				Name: "Russia",
			},
			expected: "Russia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entity.GetDisplayIdentifier(); got != tt.expected {
				t.Errorf("GetDisplayIdentifier() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEntityType(t *testing.T) {
	types := []EntityType{
		EntityTypeCountry,
		EntityTypeCity,
		EntityTypeRegion,
		EntityTypePerson,
		EntityTypeOrganization,
		EntityTypeMilitaryUnit,
		EntityTypeVessel,
		EntityTypeWeaponSystem,
		EntityTypeEvent,
		EntityTypeFacility,
		EntityTypeOther,
	}

	for _, et := range types {
		if et == "" {
			t.Errorf("EntityType should not be empty")
		}
	}
}

func TestEntityAttrs_Geographic(t *testing.T) {
	attrs := EntityAttrs{
		CountryCode: "US",
		Latitude:    38.8977,
		Longitude:   -77.0365,
	}

	if attrs.CountryCode != "US" {
		t.Error("CountryCode should be US")
	}
	if attrs.Latitude == 0 || attrs.Longitude == 0 {
		t.Error("Coordinates should be set")
	}
}

func TestEntityAttrs_Person(t *testing.T) {
	attrs := EntityAttrs{
		Title:       "President",
		Affiliation: "United States",
	}

	if attrs.Title == "" {
		t.Error("Title should be set")
	}
	if attrs.Affiliation == "" {
		t.Error("Affiliation should be set")
	}
}

func TestEntityAttrs_Military(t *testing.T) {
	attrs := EntityAttrs{
		Branch:       "Army",
		CommandLevel: "Brigade",
	}

	if attrs.Branch == "" {
		t.Error("Branch should be set")
	}
}

func TestEntityAttrs_Vessel(t *testing.T) {
	attrs := EntityAttrs{
		VesselType: "carrier",
		IMO:        "1234567",
		Flag:       "US",
	}

	if attrs.VesselType == "" {
		t.Error("VesselType should be set")
	}
	if attrs.IMO == "" {
		t.Error("IMO should be set")
	}
}

func TestEntity_FullLifecycle(t *testing.T) {
	entity := Entity{
		ID:             "ent-123",
		Type:           EntityTypeCountry,
		Name:           "United States of America",
		NormalizedName: "United States",
		Aliases:        []string{"USA", "US", "America"},
		Confidence:     0.95,
		Context:        "...in the United States...",
		Attributes: EntityAttrs{
			CountryCode: "US",
			Latitude:    38.8977,
			Longitude:   -77.0365,
			WikidataID:  "Q30",
		},
	}

	// Test primary entity
	if !entity.IsPrimaryEntity() {
		t.Error("Should be a primary entity")
	}

	// Test display identifier
	displayID := entity.GetDisplayIdentifier()
	if displayID != "United States" {
		t.Errorf("Expected 'United States', got %s", displayID)
	}

	// Test attributes
	if entity.Attributes.CountryCode != "US" {
		t.Error("CountryCode should be US")
	}
	if len(entity.Aliases) != 3 {
		t.Errorf("Expected 3 aliases, got %d", len(entity.Aliases))
	}
}

func TestEntity_ComplexMilitaryUnit(t *testing.T) {
	entity := Entity{
		ID:             "ent-mil-001",
		Type:           EntityTypeMilitaryUnit,
		Name:           "82nd Airborne Division",
		NormalizedName: "82nd Airborne Division",
		Confidence:     0.88,
		Attributes: EntityAttrs{
			Branch:       "Army",
			CommandLevel: "Division",
			Affiliation:  "United States",
		},
	}

	if !entity.IsPrimaryEntity() {
		t.Error("Military unit should be a primary entity with high confidence")
	}

	if entity.Attributes.Branch != "Army" {
		t.Error("Branch should be Army")
	}
}
