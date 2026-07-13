package eloverblik

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func ErrorClientConnection(status int) error {
	return fmt.Errorf("could't connect to eloverblik: %d", status)
}

// apiErrorBody is the error body of a failed request. Eloverblik answers with two
// different shapes and the same call can meet both, so both have to be read:
//
//   - a bare JSON string carrying the API error message, e.g. "[20010] Relation not found".
//     This is the business error the API documents and the one the error codes come with.
//   - an RFC 7807 problem document, e.g. {"type":"...","title":"Not Found","status":404,
//     "traceId":"00-9c48...-01"}. Both OpenAPI specs declare it for 400, 401, 403 and 404,
//     and it is what a request to an endpoint that is not deployed actually comes back with.
//
// Only one of the two is ever set. Unmarshalling never fails: a body that cannot be read
// is left empty and judged by its HTTP status, rather than making resty log a warning and
// drop the response on the floor.
type apiErrorBody struct {
	// Message is the API error message when the body is a bare JSON string.
	Message string
	// Problem is the problem document when the body is an object.
	Problem *problemDetails
}

// problemDetails is the RFC 7807 problem document both APIs declare as their error schema.
// TraceID is not part of the schema but is always present in practice, and is what
// Energinet support asks for when an endpoint misbehaves, so it must survive.
type problemDetails struct {
	Type     string              `json:"type"`
	Title    string              `json:"title"`
	Status   int                 `json:"status"`
	Detail   string              `json:"detail"`
	Instance string              `json:"instance"`
	TraceID  string              `json:"traceId"`
	Errors   map[string][]string `json:"errors"`
}

// UnmarshalJSON tells the two error body shapes apart by their first character: a JSON
// string opens with a quote, a problem document with a brace. Anything else is a body this
// client has no use for, and is reported by its HTTP status alone.
func (b *apiErrorBody) UnmarshalJSON(data []byte) error {
	body := bytes.TrimSpace(data)
	if len(body) == 0 {
		return nil
	}

	switch body[0] {

	case '"':
		var msg string
		if err := json.Unmarshal(body, &msg); err != nil {
			return nil
		}
		b.Message = msg

	case '{':
		var problem problemDetails
		if err := json.Unmarshal(body, &problem); err != nil {
			return nil
		}
		b.Problem = &problem
	}

	return nil
}

// APIError reports a failed request Eloverblik answered with a problem document instead of
// the usual "[code] message" string, which is what both APIs do for 400, 401, 403 and 404.
// It keeps the parts worth having: the status, the title, the detail and the trace ID -
// Energinet support asks for the trace ID, so it must reach the caller.
//
// Where the document does carry a known API error code, and for the statuses that have a
// sentinel of their own, APIError unwraps to that sentinel, so errors.Is keeps working:
//
//	var apiErr *eloverblik.APIError
//	if errors.As(err, &apiErr) {
//		log.Printf("eloverblik %d %s, traceId %s", apiErr.StatusCode, apiErr.Title, apiErr.TraceID)
//	}
type APIError struct {
	// StatusCode is the HTTP status of the response. The status the document reports takes
	// precedence over the one the response arrived with, they only ever differ if the API
	// contradicts itself.
	StatusCode int
	// Code is the API error code, e.g. 20010. Zero when the document carries none, which is
	// the usual case.
	Code uint64
	// Type is the URI the problem document identifies the problem type with.
	Type string
	// Title is the short summary of the problem, e.g. "Not Found".
	Title string
	// Detail is the explanation specific to this occurrence. Often empty.
	Detail string
	// Instance is the URI of the occurrence. Often empty.
	Instance string
	// TraceID correlates the request with Energinet's own logs. Quote it in support requests.
	TraceID string
	// Errors holds the per field validation messages of a 400. Nil unless the API sent any.
	Errors map[string][]string

	// err is the sentinel this problem unwraps to, nil when it maps to none.
	err error
}

// Error renders the problem in one line, e.g.
// "eloverblik: 404 Not Found (traceId 00-9c485a3a3ed458eab22cab724111db63-ed7aa1e057161e52-01)".
func (e *APIError) Error() string {
	var msg strings.Builder

	msg.WriteString("eloverblik: ")
	msg.WriteString(strconv.Itoa(e.StatusCode))

	if e.Title != "" {
		msg.WriteString(" " + e.Title)
	}
	if e.Detail != "" {
		msg.WriteString(": " + e.Detail)
	}
	if e.TraceID != "" {
		msg.WriteString(" (traceId " + e.TraceID + ")")
	}

	return msg.String()
}

// Unwrap returns the sentinel error the problem maps to, so a caller can keep matching on
// errors.Is(err, ErrorUnauthorized) no matter which of the two shapes the API answered with.
func (e *APIError) Unwrap() error { return e.err }

// newAPIError builds the error a problem document is reported with. statusCode is the
// status the response arrived with, and is used when the document carries none of its own.
func newAPIError(problem *problemDetails, statusCode int) *APIError {
	status := problem.Status
	if status == 0 {
		status = statusCode
	}

	apiErr := &APIError{
		StatusCode: status,
		Type:       problem.Type,
		Title:      problem.Title,
		Detail:     problem.Detail,
		Instance:   problem.Instance,
		TraceID:    problem.TraceID,
		Errors:     problem.Errors,
	}

	// A problem document has no field for an API error code, but nothing stops the API from
	// writing one into the detail. Read it when it is there, so a code keeps mapping to the
	// sentinel it always mapped to; otherwise fall back to what the status alone tells.
	if code, ok := apiErrorCode(problem.Detail); ok {
		if sentinel, known := apiErrorMap[code]; known && sentinel != nil {
			apiErr.Code = code
			apiErr.err = sentinel
			return apiErr
		}
	}
	apiErr.err = statusSentinel(status)

	return apiErr
}

// isRetryableError reports whether a response is worth retrying. Only the two transient
// conditions documented by the API qualify: 429 when a rate limit is exceeded and 503
// when DataHub is unable to keep up. Everything else - 401, any other 4xx, 500 - is a
// permanent answer and is handed to the caller right away. Transport errors are not
// retried either, so a request is never sent twice by accident.
func isRetryableError(statusCode int, err error) bool {

	if err != nil {
		return false
	}
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable
}

// isSuccessStatus reports whether a status code is a 2xx.
func isSuccessStatus(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

// statusError maps a response that carries no parseable API error message to an error
// based on its HTTP status alone.
func statusError(statusCode int) error {
	if err := statusSentinel(statusCode); err != nil {
		return err
	}
	return ErrorClientConnection(statusCode)
}

// statusSentinel returns the sentinel error a HTTP status has of its own, nil for a status
// that has none.
func statusSentinel(statusCode int) error {
	switch statusCode {
	case http.StatusTooManyRequests:
		return ErrorTooManyRequests
	case http.StatusUnauthorized:
		return ErrorUnauthorized
	default:
		return nil
	}
}

// apiErrorCode reads the API error code out of a message, e.g. 20010 out of
// "[20010] Relation not found". ok is false when the message carries no code.
func apiErrorCode(msg string) (code uint64, ok bool) {
	if len(msg) < 6 || msg[0] != '[' {
		return 0, false
	}

	code, err := strconv.ParseUint(msg[1:6], 10, 64)
	if err != nil {
		return 0, false
	}

	return code, true
}

// apiErrorFromBody turns the error body of a response into an error. It is what every
// request calls: apiErrorBody has already told the two shapes the API answers with apart,
// and each is reported in the way that keeps the most of it.
func apiErrorFromBody(body apiErrorBody, statusCode int) error {

	// A problem document only ever accompanies a failure. On a success the result body is
	// the one that matters, and resty never fills the error body in the first place.
	if body.Problem != nil && !isSuccessStatus(statusCode) {
		return newAPIError(body.Problem, statusCode)
	}

	return apiError(body.Message, statusCode)
}

func apiError(msg string, statusCode int) error {

	// Not every failure carries an API error message: a 429 from the rate limiter or a
	// 503 from DataHub can arrive with an empty or non-JSON body. Those are judged by
	// their HTTP status alone and must never be reported as success.
	if len(msg) == 0 {
		if isSuccessStatus(statusCode) {
			return apiErrorMap[10000]
		}
		return statusError(statusCode)
	}

	// API error messages carry the code in their first characters, e.g.
	// "[20010] Relation not found". A message without one holds nothing to look up.
	code, ok := apiErrorCode(msg)
	if !ok {
		return fmt.Errorf("failed to parse error in api error message %s", msg)
	}

	// API Error lookup
	errLookup, known := apiErrorMap[code]
	if !known {
		return fmt.Errorf("unhandled error: '%s'", msg)
	}

	// A mapped-to-nil code (10000, "no error") on a failed response still is a failure
	if errLookup == nil && !isSuccessStatus(statusCode) {
		return statusError(statusCode)
	}

	return errLookup
}

var (
	ErrorNoError                                        error = errors.New("no errors")                                                                     // status code 200 - api code 10000
	ErrorWrongNumberOfArguments                         error = errors.New("wrong number of arguments")                                                     // status code 400 - api code 10001
	ErrorToManyRequestItems                             error = errors.New("to many request items")                                                         // status code 412 - api code 10002
	ErrorInternalServerError                            error = errors.New("internal server error")                                                         // status code 500 - api code 10003
	ErrorMaximumNumberOfMeteringPointsExceeded          error = errors.New("number of metering point exceeds maximum of {max} metering points per request") // status code 429 - api code 10004
	ErrorNoCprConsent                                   error = errors.New("missing consent for CPR lookup")                                                // api code 10007
	ErrorWrongMeteringPointIdOrWebAccessCode            error = errors.New("invalid meteringpoint ID or webaccess code")                                    // status code 404 - api code 20000
	ErrorMeteringPointBlocked                           error = errors.New("meteringpoint blocked")                                                         // status code 403 - api code 20001
	ErrorMeteringPointAlreadyAdded                      error = errors.New("meteringpoint relation already added")                                          // status code 208 - api code 20002
	ErrorMeteringPointIdNot18CharsLong                  error = errors.New("meteringpoint ID must be 18 characters long")                                   // status code 411 - api code 20003
	ErrorMeteringpointIdContainsNonDigits               error = errors.New("meteringpoint IDs do not contain digits")                                       // status code 406 - api code 20004
	ErrorMeteringPointAliasTooLong                      error = errors.New("meteringpoint alias too long")                                                  // status code 416 - api code 20005
	ErrorWebAccessCodeNot8CharsLong                     error = errors.New("webaccess codes must be 8 characters long")                                     // status code 411 - api code 20006
	ErrorWebAccessCodeContainsIllegalChars              error = errors.New("webaccess code contains illegal characters")                                    // status code 406 - api code 20007
	ErrorMeteringPointNotFound                          error = errors.New("meteringpoint not found")                                                       // status code 404 - api code 20008
	ErrorMeteringPointIsChild                           error = errors.New("meteringpoint can't be child")                                                  // status code 422 - api code 20009
	ErrorRelationNotFound                               error = errors.New("relation not found")                                                            // status code 404 - api code 20010
	ErrorUnknownError                                   error = errors.New("unknown erro")                                                                  // status code 500 - api code 20011
	ErrorUnauthorized                                   error = errors.New("unauthorized access")                                                           // status code 401 - api code 20012
	ErrorNoValidMeteringPointsInList                    error = errors.New("no meteringpoints in request conforms to valid meteringpoint format")           // status code 400 - api code 20013
	ErrorFromDateIsGreaterThanToday                     error = errors.New("requested from date is after today")                                            // status code 400 - api code 30000
	ErrorFromDateIsGreaterThanToDate                    error = errors.New("period not allowed, ToDate is before FromDate")                                 // status code 400 - api code 30001
	ErrorToDateCanNotBeEqualToFromDate                  error = errors.New("period not allowed, ToDate is equal to FromDat")                                // status code 400 - api code 30002
	ErrorToDateIsGreaterThanToday                       error = errors.New("requested to date is after today")                                              // status code 400 - api code 30003
	ErrorInvalidDateFormat                              error = errors.New("invalid date format in request")                                                // status code 400 - api code 30004
	ErrorInvalidRequestParameters                       error = errors.New("a request parameter is invalid")                                                // status code 400 - api code 30005
	ErrorAccessToMeteringPointDenied                    error = errors.New("access to meterpoint denied")                                                   // status code 401 - api code 30006
	ErrorNoMeteringPointDataAviliable                   error = errors.New("no meterpoint data aviliable")                                                  // status code 204 - api code 30007
	ErrorRequestedAggregationUnavaliable                error = errors.New("requested data aggregation is not supported")                                   // status code 406 - api code 30008
	ErrorInvalidMeteringpointId                         error = errors.New("requested meteringpoint ID is not valid")                                       // status code 406 - api code 30009
	ErrorDateNotCoveredByAuthorization                  error = errors.New("requested date not covered by Authorization")                                   // status code 401 - api code 30010
	ErrorAggrationNotValid                              error = errors.New("requested data aggregation is not supported")                                   // status code 406 - api code 30011
	ErrorRequestToHuge                                  error = errors.New("request size too large")                                                        // status code 413 - api code 30012
	ErrorNumberOfDaysExcceded                           error = errors.New("request period exceeds the maximum number of days (730)")                       // status code 400 - api code 30014
	ErrorInvalidCVR                                     error = errors.New("CVR is invalid")                                                                // status code 403 - api code 40000
	ErrorInvalidIncludeFutureMeteringPointsRelatedToCVR error = errors.New("requested future meteringpoints related to CVR are invalid")                    // status code 404 - api code 40001
	ErrorInvalidMasterDataFields                        error = errors.New("invalid master data fields")                                                    // status code 417 - api code 40002
	ErrorInvalidMeteringPointIds                        error = errors.New("requested meteringpoint IDs are not valid")                                     // status code 406 - api code 40003
	ErrorInvalidSignature                               error = errors.New("invalid signature")                                                             // status code 403 - api code 40004
	ErrorInvalidSignedByNameId                          error = errors.New("invalid signed by name I")                                                      // status code 403 - api code 40005
	ErrorInvalidSignedDate                              error = errors.New("invalid signed date")                                                           // status code 403 - api code 40006
	ErrorInvalidSignedText                              error = errors.New("invalid signed text")                                                           // status code 403 - api code 40007
	ErrorInvalidThirdPartyId                            error = errors.New("invalid third party ID. 14/3")                                                  // status code 403 - api code 40008
	ErrorInvalidValidFrom                               error = errors.New("invalid from date")                                                             // status code 400 - api code 40009
	ErrorInvalidValidTo                                 error = errors.New("invalid to date")                                                               // status code 400 - api code 40010
	ErrorValidToBeforeValidFrom                         error = errors.New("requested from date cannot be after requested to date")                         // status code 400 - api code 40011
	ErrorValidToOutOfRange                              error = errors.New("requested to date is out of range")                                             // status code 400 - api code 40012
	ErrorValidFromOutOfRange                            error = errors.New("requested from date is out of range")                                           // status code 400 - api code 40013
	ErrorNoAuthorizationsFound                          error = errors.New("no power of attorneys found")                                                   // status code 400 - api code 40014
	ErrorWrongTokenType                                 error = errors.New("request used wrong token type")                                                 // status code 406 - api code 50000
	ErrorTokenNotValid                                  error = errors.New("token is invalid")                                                              // status code 401 - api code 50001
	ErrorErrorCreatingToken                             error = errors.New("error creating token")                                                          // status code 500 - api code 50002
	ErrorTokenRegistrationFailed                        error = errors.New("token registration failed")                                                     // status code 500 - api code 50003
	ErrorTokenAlreadyActive                             error = errors.New("token already active")                                                          // status code 405 - api code 50004
	ErrorTokenAlreadyDeactivated                        error = errors.New("token already deactived")                                                       // status code 405 - api code 50005
	ErrorTokenMissingTokenId                            error = errors.New("token do not contain a token id")                                               // status code 401 - api code 50006
	ErrorThirdPartyNotFound                             error = errors.New("third party not found")                                                         // status code 404 - api code 60000
	ErrorThirdPartyWasNotCreated                        error = errors.New("third party not created")                                                       // status code 400 - api code 60001
	ErrorThirdPartyAlreadyExist                         error = errors.New("third party already exist")                                                     // status code 208 - api code 60002
	ErrorThirdPartyApplictionInPrgress                  error = errors.New("third party application is already in progress")                                // status code 400 - api code 60004
	ErrorThirdPartyAlreadyExistButIsInactive            error = errors.New("third party already exist but is inactive")                                     // status code 401 - api code 60005
	ErrorThirdPartyAlreadyExistButIsRevoked             error = errors.New("third party already exist but access is revoked")                               // status code 401 - api code 60006
	ErrorTooManyRequests                                error = errors.New("too many requests")                                                             // status code 429
)

var apiErrorMap = map[uint64]error{
	10000: nil, //ErrorNoError,
	10001: ErrorWrongNumberOfArguments,
	10002: ErrorToManyRequestItems,
	10003: ErrorInternalServerError,
	10004: ErrorMaximumNumberOfMeteringPointsExceeded,
	10007: ErrorNoCprConsent,
	20000: ErrorWrongMeteringPointIdOrWebAccessCode,
	20001: ErrorMeteringPointBlocked,
	20002: ErrorMeteringPointAlreadyAdded,
	20003: ErrorMeteringPointIdNot18CharsLong,
	20004: ErrorMeteringpointIdContainsNonDigits,
	20005: ErrorMeteringPointAliasTooLong,
	20006: ErrorWebAccessCodeNot8CharsLong,
	20007: ErrorWebAccessCodeContainsIllegalChars,
	20008: ErrorMeteringPointNotFound,
	20009: ErrorMeteringPointIsChild,
	20010: ErrorRelationNotFound,
	20011: ErrorUnknownError,
	20012: ErrorUnauthorized,
	20013: ErrorNoValidMeteringPointsInList,
	30000: ErrorFromDateIsGreaterThanToday,
	30001: ErrorFromDateIsGreaterThanToDate,
	30002: ErrorToDateCanNotBeEqualToFromDate,
	30003: ErrorToDateIsGreaterThanToday,
	30004: ErrorInvalidDateFormat,
	30005: ErrorInvalidRequestParameters,
	30006: ErrorAccessToMeteringPointDenied,
	30007: ErrorNoMeteringPointDataAviliable,
	30008: ErrorRequestedAggregationUnavaliable,
	30009: ErrorInvalidMeteringpointId,
	30010: ErrorDateNotCoveredByAuthorization,
	30011: ErrorAggrationNotValid,
	30012: ErrorRequestToHuge,
	30014: ErrorNumberOfDaysExcceded,
	40000: ErrorInvalidCVR,
	40001: ErrorInvalidIncludeFutureMeteringPointsRelatedToCVR,
	40002: ErrorInvalidMasterDataFields,
	40003: ErrorInvalidMeteringPointIds,
	40004: ErrorInvalidSignature,
	40005: ErrorInvalidSignedByNameId,
	40006: ErrorInvalidSignedDate,
	40007: ErrorInvalidSignedText,
	40008: ErrorInvalidThirdPartyId,
	40009: ErrorInvalidValidFrom,
	40010: ErrorInvalidValidTo,
	40011: ErrorValidToBeforeValidFrom,
	40012: ErrorValidToOutOfRange,
	40013: ErrorValidFromOutOfRange,
	40014: ErrorNoAuthorizationsFound,
	50000: ErrorWrongTokenType,
	50001: ErrorTokenNotValid,
	50002: ErrorErrorCreatingToken,
	50003: ErrorTokenRegistrationFailed,
	50004: ErrorTokenAlreadyActive,
	50005: ErrorTokenAlreadyDeactivated,
	50006: ErrorTokenMissingTokenId,
	60000: ErrorThirdPartyNotFound,
	60001: ErrorThirdPartyWasNotCreated,
	60002: ErrorThirdPartyAlreadyExist,
	60004: ErrorThirdPartyApplictionInPrgress,
	60005: ErrorThirdPartyAlreadyExistButIsInactive,
	60006: ErrorThirdPartyAlreadyExistButIsRevoked,
}
