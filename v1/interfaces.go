package eloverblik

import "time"

type Client interface {
	GetDataAccessToken() (string, error)
	// GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetails, error)
	// GetCharges(meteringPointIDs []string) ([]MeteringPointPrices, error)
	GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error)
	GetMeterReadings(meteringPointIDs []string, from, to time.Time) ([]MeterReading, error)
}

type Customer interface {
	Client
	// AddRelationOnID(meteringPointIDs []string) error
	// AddRelationOnAccessCode(meteringPointID, webAccessCode string) error
	// DeleteRelation(meteringPointID string) error
	GetMeteringPoints(includeAll bool) ([]MeteringPoints, error)
}

type ThirdParty interface {
	Client
	// GetAuthorizations() error
	// GetMeteringPointsForScope(scope, identifier string) ([]MeteringPoints, error)
}
