package eloverblik

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

func ErrorClientConnection(status int) error {
	return fmt.Errorf("could't connect to eloverblik: %d", status)
}

func isRetryableError(status int, err error) bool {

	if err == nil || status == http.StatusOK {
		return false
	}
	return true
}

func apiError(msg string, statusCode int) error {

	if len(msg) == 0 {
		return apiErrorMap[10000]
	}

	// Parse API error code
	code, err := strconv.ParseUint(msg[1:6], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse error in api error message %s", msg)
	}

	// API Error lookup
	errLookup := apiErrorMap[code]
	if errLookup == nil && statusCode == 400 {
		return fmt.Errorf("unhandled error: '%s'", msg)
	}

	return errLookup
}

var (
	ErrorNoError                                        error = errors.New("no errors")                                                                     // status code 200 - api code 10000
	ErrorWrongNumberOfArguments                         error = errors.New("wrong number of arguments")                                                     // status code 400 - api code 10001
	ErrorToManyRequestItems                             error = errors.New("to many request items")                                                         // status code 412 - api code 10002
	ErrorInternalServerError                            error = errors.New("internal server error")                                                         // status code 500 - api code 10003
	ErrorMaximumNumberOfMeteringPointsExceeded          error = errors.New("number of metering point exceeds maximum of {max} metering points per request") // status code 429 - api code 10004
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
