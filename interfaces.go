package eloverblik

type CustomerAPI interface {
	GetDataAccessToken() (string, error)
	AddRelationOnID() error
	AddRelationOnAccessCode() error
	DeleteRelation() error
	GetMeteringPoints() error
	GetMeteringPointDetails() error
	GetCharges() error
	GetTimeSeries() error
	GetMeterReadings() error
}

type ThirdPartyAPI interface {
	GetDataAccessToken() (string, error)
	GetAuthorizations() error
	GetMeteringPoints() error
	GetMeteringPointDetails() error
	GetCharges() error
	GetTimeSeries() error
	GetMeterReadings() error
}
