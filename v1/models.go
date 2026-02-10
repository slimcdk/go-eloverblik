package eloverblik

import (
	"strings"
	"time"
)

// FlexibleTime handles JSON date fields that may be empty strings, null, or valid timestamps
type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = strings.Trim(s, `"`)

	// Handle empty string or null
	if s == "" || s == "null" {
		ft.Time = time.Time{}
		return nil
	}

	// Try parsing as RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	ft.Time = t
	return nil
}

func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + ft.Time.Format(time.RFC3339) + `"`), nil
}

type meteringPointIDs struct {
	MeteringPointID meteringPointID `json:"meteringPoints"`
}

type meteringPointID struct {
	MeteringPointIDs []string `json:"meteringPoint"`
}

type StatusResponse struct {
	Success    bool   `json:"success"`
	ErrorCode  int    `json:"errorCode"`
	ErrorText  string `json:"errorText"`
	ID         string `json:"id"`
	StackTrace string `json:"stackTrace"`
}

type TimeseriesError struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Status  int    `json:"status"`
	TraceID string `json:"traceId"`
	Errors  struct {
		DollarSign []string `json:"$"`
	} `json:"errors"`
}

type StringResponse struct {
	Result string `json:"result,omitempty"`
	StatusResponse
}
