package models

import (
	"testing"
	"time"
)

func TestEventQuery_Validate(t *testing.T) {
	tests := []struct {
		name          string
		query         EventQuery
		expectedPage  int
		expectedLimit int
		expectedSort  EventSortField
		expectedOrder SortOrder
	}{
		{
			name:          "Empty query gets defaults",
			query:         EventQuery{},
			expectedPage:  1,
			expectedLimit: 20,
			expectedSort:  SortByTimestamp,
			expectedOrder: SortOrderDesc,
		},
		{
			name: "Custom values preserved",
			query: EventQuery{
				Page:      5,
				Limit:     50,
				SortBy:    SortByMagnitude,
				SortOrder: SortOrderAsc,
			},
			expectedPage:  5,
			expectedLimit: 50,
			expectedSort:  SortByMagnitude,
			expectedOrder: SortOrderAsc,
		},
		{
			name: "Zero page becomes 1",
			query: EventQuery{
				Page: 0,
			},
			expectedPage:  1,
			expectedLimit: 20,
			expectedSort:  SortByTimestamp,
			expectedOrder: SortOrderDesc,
		},
		{
			name: "Negative page becomes 1",
			query: EventQuery{
				Page: -5,
			},
			expectedPage:  1,
			expectedLimit: 20,
			expectedSort:  SortByTimestamp,
			expectedOrder: SortOrderDesc,
		},
		{
			name: "Zero limit becomes 20",
			query: EventQuery{
				Limit: 0,
			},
			expectedPage:  1,
			expectedLimit: 20,
			expectedSort:  SortByTimestamp,
			expectedOrder: SortOrderDesc,
		},
		{
			name: "Limit capped at 1000",
			query: EventQuery{
				Limit: 1500,
			},
			expectedPage:  1,
			expectedLimit: 1000,
			expectedSort:  SortByTimestamp,
			expectedOrder: SortOrderDesc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if err != nil {
				t.Errorf("Validate() returned error: %v", err)
			}
			if tt.query.Page != tt.expectedPage {
				t.Errorf("Page = %v, want %v", tt.query.Page, tt.expectedPage)
			}
			if tt.query.Limit != tt.expectedLimit {
				t.Errorf("Limit = %v, want %v", tt.query.Limit, tt.expectedLimit)
			}
			if tt.query.SortBy != tt.expectedSort {
				t.Errorf("SortBy = %v, want %v", tt.query.SortBy, tt.expectedSort)
			}
			if tt.query.SortOrder != tt.expectedOrder {
				t.Errorf("SortOrder = %v, want %v", tt.query.SortOrder, tt.expectedOrder)
			}
		})
	}
}

func TestEventQuery_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		limit    int
		expected int
	}{
		{"Page 1", 1, 20, 0},
		{"Page 2", 2, 20, 20},
		{"Page 3", 3, 20, 40},
		{"Page 1 with limit 50", 1, 50, 0},
		{"Page 5 with limit 10", 5, 10, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := EventQuery{
				Page:  tt.page,
				Limit: tt.limit,
			}
			if got := q.GetOffset(); got != tt.expected {
				t.Errorf("GetOffset() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventQuery_WithFilters(t *testing.T) {
	now := time.Now()
	since := now.Add(-24 * time.Hour)
	minMag := 5.0
	minConf := 0.7

	query := EventQuery{
		SearchQuery:    "Ukraine",
		SinceTimestamp: &since,
		MinMagnitude:   &minMag,
		MinConfidence:  &minConf,
		Categories:     []Category{CategoryGeopolitics, CategoryMilitary},
		SourceTypes:    []SourceType{SourceTypeTwitter, SourceTypeTelegram},
		Tags:           []string{"conflict", "breaking"},
		Page:           1,
		Limit:          50,
	}

	if err := query.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	if query.SearchQuery != "Ukraine" {
		t.Error("SearchQuery should be preserved")
	}
	if query.SinceTimestamp == nil || !query.SinceTimestamp.Equal(since) {
		t.Error("SinceTimestamp should be preserved")
	}
	if *query.MinMagnitude != 5.0 {
		t.Error("MinMagnitude should be preserved")
	}
	if *query.MinConfidence != 0.7 {
		t.Error("MinConfidence should be preserved")
	}
	if len(query.Categories) != 2 {
		t.Error("Categories should be preserved")
	}
	if len(query.SourceTypes) != 2 {
		t.Error("SourceTypes should be preserved")
	}
	if len(query.Tags) != 2 {
		t.Error("Tags should be preserved")
	}
}

func TestEventSortField(t *testing.T) {
	fields := []EventSortField{
		SortByTimestamp,
		SortByMagnitude,
		SortByConfidence,
		SortByCreatedAt,
		SortByUpdatedAt,
	}

	for _, field := range fields {
		if field == "" {
			t.Errorf("EventSortField should not be empty")
		}
	}
}

func TestSortOrder(t *testing.T) {
	if SortOrderAsc == "" || SortOrderDesc == "" {
		t.Error("SortOrder constants should not be empty")
	}
}

func TestEventResponse(t *testing.T) {
	events := []Event{
		{ID: "evt-1", Title: "Event 1"},
		{ID: "evt-2", Title: "Event 2"},
	}

	response := EventResponse{
		Events:  events,
		Page:    1,
		Limit:   20,
		Total:   50,
		HasMore: true,
		Query:   "test query",
	}

	if len(response.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(response.Events))
	}
	if !response.HasMore {
		t.Error("HasMore should be true")
	}
	if response.Total != 50 {
		t.Errorf("Expected total 50, got %d", response.Total)
	}
}

func TestEventQuery_Pagination(t *testing.T) {
	query := EventQuery{
		Page:  3,
		Limit: 25,
	}

	if err := query.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	offset := query.GetOffset()
	expectedOffset := (3 - 1) * 25 // 50

	if offset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, offset)
	}
}
