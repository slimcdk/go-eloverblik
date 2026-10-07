package eloverblik

import (
	"io"
	"time"
)

type Client interface {
	// GetDataAccessToken returns the data access token the client sends on its requests.
	// The client fetches it lazily, exchanging its refresh token for one at /token on the
	// first call that needs it, and caches it.
	//
	// A data access token lasts about 24 hours. Once the cached one has expired, or expires
	// within five minutes, according to its exp claim, the next call fetches a new one, so
	// a long running client keeps working. A token whose expiry cannot be read, because it
	// is not a JWT or its exp is missing, null, zero or negative, is kept.
	//
	// It is safe for concurrent use. The API allows only 2 /token calls a minute, so the
	// goroutines that need a token while one is being fetched wait for that request and
	// share its outcome, its token or its error. A call made after a request has failed
	// sends a new one.
	//
	// An error means the client holds no data access token that works: /token failed, and
	// there is no cached token or the cached one has expired. It is the error the request
	// failed with, e.g. ErrorTokenNotValid when the refresh token has expired or been
	// revoked, or ErrorTooManyRequests when the /token rate limit is spent. A renewal that
	// fails before the cached token has expired is not an error: the call returns the
	// cached token, which still works for a few minutes, and the next call tries again.
	GetDataAccessToken() (string, error)
	RefreshTokenClaims() (TokenClaims, error)
	// DataAccessTokenClaims decodes the claims of the data access token GetDataAccessToken
	// returns. Like any call that needs the token, it may fetch one first, or renew the
	// cached one when it has expired or expires within five minutes.
	DataAccessTokenClaims() (TokenClaims, error)
	GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetailsResponse, error)
	GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error)
	GetChargeLinksWithCharges(meteringPointIDs []string, from, to time.Time) (*ChargeLinksWithChargesResponse, error)
	IsAlive() (bool, error)
}

type Customer interface {
	Client
	GetCustomerCharges(meteringPointIDs []string) ([]CustomerChargeResponse, error)
	AddRelationByID(meteringPointIDs []string) ([]StringResponse, error)

	// AddRelationByWebAccessCode linked a metering point to the user by its web access code.
	//
	// Deprecated: Energinet retired the endpoint with DataHub 3.0. It answers 410 Gone, which
	// is reported as ErrorEndpointRetired. Data is shared through ElOverblik instead, see
	// https://docs.eloverblik.dk/docs/guides/data-sharing.
	AddRelationByWebAccessCode(meteringPointID, webAccessCode string) (string, error)

	// DeleteRelation deleted the user's relation to a metering point.
	//
	// Deprecated: Energinet retired the endpoint with DataHub 3.0. It answers 410 Gone, which
	// is reported as ErrorEndpointRetired, and no longer deletes the relation.
	DeleteRelation(meteringPointID string) (bool, error)

	GetMeteringPoints(includeAll bool) ([]MeteringPoints, error)
	// ExportTimeSeries exports the time series of the metering points as the CSV file
	// Eloverblik generates: separated by semicolons, opening with a UTF-8 byte order mark,
	// with Danish column names. The caller closes the returned stream.
	//
	// Unlike GetTimeSeries, the export includes to: it covers the Copenhagen days from
	// from through to. A to equal to from is rejected with
	// ErrorToDateCanNotBeEqualToFromDate, so an export covers at least two days. For the
	// days GetTimeSeries(ids, from, to, aggregation) returns, pass to.AddDate(0, 0, -1).
	// Like the other exports, the MålepunktsID column holds each ID with a tab before it.
	// Both were checked against the live API on 2026-10-07.
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
