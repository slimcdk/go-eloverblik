package eloverblik

import (
	"io"
	"time"
)

type Client interface {
	GetDataAccessToken() (string, error)
	GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetailsResponse, error)
	GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error)
	IsAlive() (bool, error)
}

type Customer interface {
	Client
	GetCustomerCharges(meteringPointIDs []string) ([]CustomerChargeResponse, error)
	AddRelationByID(meteringPointIDs []string) ([]StringResponse, error)
	AddRelationByWebAccessCode(meteringPointID, webAccessCode string) (string, error)
	DeleteRelation(meteringPointID string) (bool, error)
	GetMeteringPoints(includeAll bool) ([]MeteringPoints, error)
	ExportTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) (io.ReadCloser, error)
	ExportMasterdata(meteringPointIDs []string) (io.ReadCloser, error)
	ExportCharges(meteringPointIDs []string) (io.ReadCloser, error)
}

type ThirdParty interface {
	Client
	GetThirdPartyCharges(meteringPointIDs []string) ([]ThirdPartyChargeResponse, error)
	GetAuthorizations() ([]Authorization, error)
	GetMeteringPointsForScope(scope AuthorizationScope, identifier string) ([]ThirdPartyMeteringPoint, error)
	GetMeteringPointIDsForScope(scope AuthorizationScope, identifier string) ([]string, error)
}
