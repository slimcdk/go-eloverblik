// Package eloverblik is a client for Eloverblik, Energinet's API to the electricity
// metering data DataHub holds for Denmark: the metering points (målepunkter) a person or
// a company has, their master data, their consumption and production time series, and
// the charges (subscriptions, fees and tariffs) they are billed with.
//
//	import eloverblik "github.com/slimcdk/go-eloverblik/v1"
//
// The full reference, with every method, type and error code, the CLI, and what the live
// API actually answers, is llms.md:
// https://github.com/slimcdk/go-eloverblik/blob/master/llms.md. The README covers
// installation, the CLI and how to get a token: https://github.com/slimcdk/go-eloverblik.
//
// # Clients
//
// Eloverblik has two APIs, and a client for each:
//
//   - [NewCustomer] returns a [Customer], for the Customer API: a person's or a
//     company's own metering points. Its refresh token is created at
//     https://eloverblik.dk under "API-adgang", after logging in with MitID.
//   - [NewThirdParty] returns a [ThirdParty], for the Third-Party API: the metering
//     points of the customers who have given the third party a power of attorney
//     (fuldmagt). Its refresh token is created the same way, logged in with MitID
//     Erhverv.
//
// Both embed [Client], the calls the two APIs have in common, and take [Option] values:
// [WithRetry] and [WithoutRetry] set the retry policy, [WithResponseHeaderOutput] writes
// the response headers of every call to an io.Writer for debugging. Both always call the
// production API at api.eloverblik.dk. The package variables Mode, TestMode, ReleaseMode
// and ApiType are deprecated and have no effect.
//
// # Tokens
//
// A refresh token is long lived, typically a year. The client exchanges it at /token
// for a data access token, which lasts about 24 hours and is what every call but
// IsAlive sends. It does so lazily, on the first call that needs a data access token,
// caches the token, and fetches a new one once the cached one has expired or expires
// within five minutes, so a long running process can keep the same client.
//
// A client is safe for concurrent use by multiple goroutines. The API allows only 2
// /token calls a minute per IP address, so create one client per refresh token and
// share it: the goroutines that need a data access token while one is being fetched
// wait for that request and share its outcome. When /token fails, a call that needs the
// token fails with its error, e.g. [ErrorTokenNotValid] when the refresh token has
// expired or been revoked, unless the cached token has not expired yet: the client then
// keeps using it, and tries /token again on the next call.
//
// [ParseToken], [Client.RefreshTokenClaims] and [Client.DataAccessTokenClaims] decode
// the claims of a token, such as its expiry, the API it was issued for and its roles,
// without verifying its signature. RefreshTokenClaims sends no request.
//
// # Calls by task
//
// Customer API:
//
//   - The metering points: [Customer.GetMeteringPoints]. With includeAll set to true it
//     also returns the ones registered in DataHub to the user's CPR or CVR number that
//     are not linked to the user yet; [Customer.AddRelationByID] links them.
//   - Master data: [Client.GetMeteringPointDetails].
//   - Consumption and production: [Client.GetTimeSeries].
//   - Charges valid now or in the future: [Customer.GetCustomerCharges].
//   - CSV files: [Customer.ExportTimeSeries], [Customer.ExportMasterdata] and
//     [Customer.ExportCharges].
//
// Third-Party API:
//
//   - The powers of attorney: [ThirdParty.GetAuthorizations].
//   - The metering points of one: [ThirdParty.GetMeteringPointIDsForScope] with
//     [AuthScopeID] and its [Authorization] ID. [ThirdParty.GetMeteringPointsForScope]
//     returns them with their addresses, and the scopes [AuthScopeCustomerCVR] and
//     [AuthScopeCustomerKey] select a customer instead of an authorization.
//   - Master data: [Client.GetMeteringPointDetails].
//   - Consumption and production: [Client.GetTimeSeries].
//   - Charges valid now or in the future: [ThirdParty.GetThirdPartyCharges].
//
// Both: [Client.GetChargeLinksWithCharges] for historic prices, which answers 404 today
// (see Retired and unavailable endpoints), and [Client.IsAlive], which needs no token.
//
// # Dates
//
// [Client.GetTimeSeries] and [Customer.ExportTimeSeries] take from and to as time.Time,
// but send only their calendar dates in Copenhagen time (Europe/Copenhagen), whatever
// zone the values are in. The time of day is dropped, and a time near midnight in
// another zone can land on the neighbouring Danish date, so build the bounds as
// midnights in Europe/Copenhagen. The package embeds the time zone database, so
// time.LoadLocation("Europe/Copenhagen") works on every platform.
//
// The range is half-open, [from, to): to is excluded, so to include a last day D, pass
// D plus one day. The API rejects a range whose two dates are equal, with error 30002
// ([ErrorToDateCanNotBeEqualToFromDate]), and one longer than 730 days
// ([MaximumDayRequestLeap]), with error 30014 ([ErrorPeriodNotAllowed]). The client
// checks neither, so split a longer range yourself.
//
// [Client.GetChargeLinksWithCharges] is the exception: it sends both bounds as
// timestamps in Copenhagen time and keeps the time of day, so pass Copenhagen midnights
// to ask for whole days.
//
// [GetDatesFromPeriod] returns the bounds of a named [Period], computed in Copenhagen
// time: [Yesterday], [ThisWeek], [LastWeek], [ThisMonth], [LastMonth], [ThisYear] or
// [LastYear]. Weeks run from Monday to Sunday. Its to is exclusive too: 00:00 on the
// first day after the period, or now for the this_* periods, so a date-based call stops
// before today, which is not complete. On the first day of a this_* period, a Monday,
// the 1st of the month or 1 January, from and to would fall on the same date, so it
// returns an error wrapping [ErrorPeriodHasNoCompleteDay] instead.
//
// # Batches and per metering point failures
//
// The calls that take a []string of metering point IDs send them all in one request.
// Energinet asks for at most 10 metering points per request. The client does not
// enforce it, so split a longer list yourself, e.g. with slices.Chunk(ids, 10).
//
// Such a call returns an error only when the request fails as a whole. A metering point
// that fails on its own, e.g. with 30018 because the period lies outside its data, is
// reported in its own result and nowhere else. Every result of
// [Client.GetMeteringPointDetails], [Client.GetTimeSeries],
// [Customer.GetCustomerCharges], [ThirdParty.GetThirdPartyCharges] and
// [Customer.AddRelationByID] embeds [StatusResponse], whose [StatusResponse.Err]
// returns the failure, nil on success. Check Err before using a result: a failed one
// holds no data, so a failed time series flattens to no points rather than to an error.
// Each result carries its metering point ID. Match results to metering points by that
// ID, never by position, and expect an ID more than once: the API can report a metering
// point once per access period, so collect every result under its ID rather than
// overwrite an earlier one:
//
//	for _, ts := range series {
//		if err := ts.Err(); err != nil {
//			log.Printf("skipping: %v", err)
//			continue
//		}
//		points := ts.Flatten()
//		// ...
//	}
//
// [ChargeLinksWithChargesResult] has no StatusResponse; it reports a failure in its
// Error field instead.
//
// # Errors
//
// A non-2xx response is always an error, except for [Client.IsAlive], which reports it
// as false. Most failures map to the exported Error* sentinels. Match them with
// errors.Is rather than ==, as some reach the caller wrapped:
//
//   - An API error message, e.g. "[20010] Relation not found", maps to the sentinel of
//     its code, here [ErrorRelationNotFound], whatever the HTTP status.
//   - An RFC 7807 problem document, which both OpenAPI documents declare for 400, 401,
//     403 and 404, is an [*APIError]: get it with errors.AsType or errors.As for its
//     StatusCode, Title, Detail and TraceID, the ID Energinet support asks for. It
//     unwraps to the sentinel of an API error code in its detail, or else to the
//     sentinel of its status, for the statuses below that have one.
//   - A response without a usable body is judged by its status: [ErrorUnauthorized] for
//     401, [ErrorEndpointRetired] for 410 and [ErrorTooManyRequests] for 429. Any other
//     status is an [ErrorClientConnection] error, which no sentinel matches.
//   - The error of [StatusResponse.Err] unwraps to the sentinel of its code.
//
// An API error code the client has no sentinel for is still an error, one that matches
// only the sentinel of its HTTP status, if the status has one. A transport error is
// returned as it is, except when the request of an export itself fails, which wraps it
// (see Exports); a failed /token exchange is returned unwrapped by every call.
//
// # Time series
//
// [Client.GetTimeSeries] returns a [TimeSeries] per metering point, which holds the
// API's market document. [TimeSeries.Flatten] turns it into a list of
// [FlatTimeSeriesPoint], one per point: the interval [From, To) it covers, in
// Copenhagen time, its Measurement, Unit (e.g. KWH), Quality and Resolution.
//
// The [Aggregation] requested, [Actual], [Quarter], [Hour], [Day], [Month] or [Year], is
// not the [Resolution] that comes back. The live API sends PT1D and PT1Y for Day and
// Year, where its OpenAPI documents name them P1D and P1Y and add PXD. Flatten accepts
// both spellings, as well as PT15M, PT1H and P1M. A period that holds a single point,
// which is every Day, Month and Year period, gives that point the interval the API
// states, which may be partial (a Year period can cover April to December). In a period
// of several points, each point's interval is worked out from the resolution: in elapsed
// time for PT15M and PT1H, so the 23 or 25 hours of a daylight saving day land at their
// true instants, in calendar units for days, months and years, and spread evenly over
// the period for PXD.
//
// Quality is a code from the API: A04 measured, A03 estimated, A02 not available, and
// A05 incomplete. An A02 point carries no quantity, so its Measurement flattens to 0: a
// 0 is a reading only when Quality is not A02.
//
// # Retries and rate limits
//
// The API allows 2 calls a minute to /token and 120 calls a minute in all per IP
// address, and 1200 calls a minute across all users. It answers 429 when a limit is
// exceeded, and 503 when DataHub cannot keep up. The client retries those two statuses
// and nothing else: [DefaultRetryCount] (2) times, with a jittered backoff from
// [DefaultRetryWait], honouring a Retry-After header up to [DefaultRetryMaxWait] (60s).
// Any other status, such as a 401 or a 500, and a transport error are returned at once.
// [WithRetry] sets the count and the longest wait, and [WithoutRetry] turns retrying
// off.
//
// # Retired and unavailable endpoints
//
// Energinet retired two Customer API endpoints with DataHub 3.0:
// [Customer.AddRelationByWebAccessCode] and [Customer.DeleteRelation] are deprecated,
// and fail with [ErrorEndpointRetired], as the endpoints answer 410 Gone.
//
// [Client.GetChargeLinksWithCharges] is the endpoint for historic prices, where
// GetCustomerCharges and GetThirdPartyCharges return only the charges valid now or in
// the future. Both OpenAPI documents declare it, but Energinet has not enabled it:
// checked on 2026-07-13, it answered 404 on both APIs, with a problem document, which
// the client reports as an [*APIError] with StatusCode 404. Do not depend on it
// returning data.
//
// # Exports
//
// [Customer.ExportTimeSeries], [Customer.ExportMasterdata] and [Customer.ExportCharges]
// return the CSV file the API sends as an io.ReadCloser, streamed as it arrives rather
// than read into memory. The caller must close it. When the export request fails, the
// error is wrapped as "failed to export <what>: ...", and the client closes the body
// itself.
package eloverblik
