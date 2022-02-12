package eloverblik

import "time"

type BaseAPI interface {
	GetDataAccessToken() (string, error)
	GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetails, error)
	GetCharges(meteringPointIDs []string) ([]MeteringPointPrices, error)
	GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error)
	GetMeterReadings(meteringPointIDs []string, from, to time.Time) error
}

type CustomerAPI interface {
	BaseAPI
	AddRelationOnID(meteringPointIDs []string) error
	AddRelationOnAccessCode(meteringPointID, webAccessCode string) error
	DeleteRelation(meteringPointID string) error
	GetMeteringPoints(bool) ([]MeteringPoints, error)
}

type ThirdPartyAPI interface {
	BaseAPI
	GetAuthorizations() error
	GetMeteringPoints(scope, identifier string) ([]MeteringPoints, error)
}
