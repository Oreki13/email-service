package dto

import (
	"email-service/internal/domain"
	"time"
)

// EmailTrackingResponseType adalah response yang berisi data tracking email
type EmailTrackingResponseType struct {
	ID        string              `json:"id"`
	EmailID   string              `json:"email_id"`
	Type      domain.TrackingType `json:"type"`
	Timestamp time.Time           `json:"timestamp"`
	UserAgent string              `json:"user_agent,omitempty"`
	IPAddress string              `json:"ip_address,omitempty"`
	URL       string              `json:"url,omitempty"`
	Count     int                 `json:"count"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// EmailTrackingStatsResponseType adalah response yang berisi statistik tracking email
type EmailTrackingStatsResponseType struct {
	EmailID    string                      `json:"email_id"`
	OpenCount  int                         `json:"open_count"`
	ClickCount int                         `json:"click_count"`
	ClickURLs  map[string]int              `json:"click_urls,omitempty"`
	History    []EmailTrackingResponseType `json:"history,omitempty"`
}

// EmailTrackingResponse membuat instance baru EmailTrackingResponse dari domain model
func EmailTrackingResponse(tracking *domain.EmailTracking) EmailTrackingResponseType {
	return EmailTrackingResponseType{
		ID:        tracking.ID,
		EmailID:   tracking.EmailID,
		Type:      tracking.Type,
		Timestamp: tracking.Timestamp,
		UserAgent: tracking.UserAgent,
		IPAddress: tracking.IPAddress,
		URL:       tracking.URL,
		Count:     tracking.Count,
		CreatedAt: tracking.CreatedAt,
		UpdatedAt: tracking.UpdatedAt,
	}
}

// EmailTrackingStatsResponse membuat instance baru EmailTrackingStatsResponse
func EmailTrackingStatsResponse(emailID string, trackingData []*domain.EmailTracking) EmailTrackingStatsResponseType {
	response := EmailTrackingStatsResponseType{
		EmailID:    emailID,
		OpenCount:  0,
		ClickCount: 0,
		ClickURLs:  make(map[string]int),
		History:    make([]EmailTrackingResponseType, 0, len(trackingData)),
	}

	// Hitung statistik dan konversi ke response object
	for _, tracking := range trackingData {
		trackingResponse := EmailTrackingResponse(tracking)
		response.History = append(response.History, trackingResponse)

		if tracking.Type == domain.TrackingTypeOpen {
			response.OpenCount += tracking.Count
		} else if tracking.Type == domain.TrackingTypeClick {
			response.ClickCount += tracking.Count
			response.ClickURLs[tracking.URL] += tracking.Count
		}
	}

	return response
}
