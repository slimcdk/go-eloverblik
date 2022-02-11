package eloverblik

type CustomerAPI interface {
	GetDataAccessToken() (string, error)
	AddRelationOnID() error
	AddRelationOnAccessCode() error
	DeleteRelation() error
	GetMeteringPoints(bool) ([]MeteringPoints, error)
	GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetails, error)
	GetCharges() error
	GetTimeSeries(query TimeseriesQuery) ([]TimeSeriesResponse, error)
	GetMeterReadings() error
}

type ThirdPartyAPI interface {
	GetDataAccessToken() (string, error)
	GetAuthorizations() error
	GetMeteringPoints(bool) ([]MeteringPoints, error)
	GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetails, error)
	GetCharges() error
	GetTimeSeries(query TimeseriesQuery) ([]TimeSeriesResponse, error)
	GetMeterReadings() error
}
