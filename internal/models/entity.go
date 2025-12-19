package models

// Entity represents a named entity extracted from OSINT content (e.g., countries, persons, organizations).
type Entity struct {
	ID             string      `json:"id"`
	Type           EntityType  `json:"type"`
	Name           string      `json:"name"`
	NormalizedName string      `json:"normalized_name"` // Canonical form for deduplication
	Aliases        []string    `json:"aliases,omitempty"`
	Confidence     float64     `json:"confidence"` // 0-1 NLP extraction confidence
	Context        string      `json:"context"`    // Surrounding text snippet
	Attributes     EntityAttrs `json:"attributes,omitempty"`
}

// EntityType categorizes extracted named entities.
type EntityType string

const (
	EntityTypeCountry      EntityType = "country"
	EntityTypeCity         EntityType = "city"
	EntityTypeRegion       EntityType = "region"
	EntityTypePerson       EntityType = "person"
	EntityTypeOrganization EntityType = "organization"
	EntityTypeMilitaryUnit EntityType = "military_unit"
	EntityTypeVessel       EntityType = "vessel"
	EntityTypeWeaponSystem EntityType = "weapon_system"
	EntityTypeEvent        EntityType = "event"
	EntityTypeFacility     EntityType = "facility"
	EntityTypeOther        EntityType = "other"
)

// EntityAttrs holds type-specific attributes for different entity types.
type EntityAttrs struct {
	// Geographic entities
	CountryCode string  `json:"country_code,omitempty"` // ISO 3166-1 alpha-2
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`

	// Person entities
	Title       string `json:"title,omitempty"`       // e.g., "President", "General"
	Affiliation string `json:"affiliation,omitempty"` // Organization/country affiliation

	// Organization entities
	OrgType      string `json:"org_type,omitempty"` // e.g., "military", "government", "ngo"
	Headquarters string `json:"headquarters,omitempty"`

	// Military entities
	Branch       string `json:"branch,omitempty"`        // e.g., "Army", "Navy", "Air Force"
	CommandLevel string `json:"command_level,omitempty"` // e.g., "Brigade", "Battalion"

	// Vessel entities
	VesselType string `json:"vessel_type,omitempty"` // e.g., "carrier", "submarine", "cargo"
	IMO        string `json:"imo,omitempty"`         // International Maritime Organization number
	Flag       string `json:"flag,omitempty"`        // Flag state

	// Weapon system entities
	WeaponClass string `json:"weapon_class,omitempty"` // e.g., "missile", "aircraft", "tank"
	Designation string `json:"designation,omitempty"`  // e.g., "S-400", "F-35"

	// Reference data
	WikidataID   string   `json:"wikidata_id,omitempty"`   // For knowledge graph linking
	ExternalRefs []string `json:"external_refs,omitempty"` // Additional reference URLs
}

// IsPrimaryEntity returns true if this is a high-confidence, core entity type.
func (e *Entity) IsPrimaryEntity() bool {
	primaryTypes := map[EntityType]bool{
		EntityTypeCountry:      true,
		EntityTypeCity:         true,
		EntityTypePerson:       true,
		EntityTypeOrganization: true,
		EntityTypeMilitaryUnit: true,
	}
	return primaryTypes[e.Type] && e.Confidence >= 0.7
}

// GetDisplayIdentifier returns the best identifier for display purposes.
func (e *Entity) GetDisplayIdentifier() string {
	if e.NormalizedName != "" {
		return e.NormalizedName
	}
	return e.Name
}
