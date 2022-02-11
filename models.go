package eloverblik

import (
	"net/http"
	"net/url"
)

type client struct {
	refreshToken string
	accessToken  string

	hostUrl url.URL
	client  *http.Client
}

type meteringPointIDs struct {
	MeteringPointID meteringPointID `json:"meteringPoints"`
}

type meteringPointID struct {
	MeteringPointIDs []string `json:"meteringPoint"`
}

type StatusResponse struct {
	Success    bool   `json:"success"`
	ErrorCode  string `json:"errorCode"`
	ErrorText  string `json:"errorText"`
	ID         string `json:"id"`
	StackTrace string `json:"stackTrace"`
}

type meterReadingResult struct {
	Result []MeterReadingResponse `json:"result"`
}

type MeterReadingResponse struct {
	Result MeterReadingResponseResult `json:"result"`
}

type MeterReadingResponseResult struct {
	//`json:"meteringPointId"`
}
