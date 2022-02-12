package eloverblik

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrorNoError                                        error = errors.New("no errors")                                                                     // status code  200
	ErrorWrongNumberOfArguments                         error = errors.New("wrong number of arguments")                                                     // status code  400
	ErrorToManyRequestItems                             error = errors.New("to many request items")                                                         // status code  412
	ErrorInternalServerError                            error = errors.New("internal server error")                                                         // status code  500
	ErrorMaximumNumberOfMeteringPointsExceeded          error = errors.New("number of metering point exceeds maximum of {max} metering points per request") // status code  429
	ErrorWrongMeteringPointIdOrWebAccessCode            error = errors.New("invalid meteringpoint ID or webaccess code")                                    // status code  404
	ErrorMeteringPointBlocked                           error = errors.New("meteringpoint blocked")                                                         // status code  403
	ErrorMeteringPointAlreadyAdded                      error = errors.New("meteringpoint relation already added")                                          // status code  208
	ErrorMeteringPointIdNot18CharsLong                  error = errors.New("meteringpoint ID must be 18 characters long")                                   // status code  411
	ErrorMeteringpointIdContainsNonDigits               error = errors.New("meteringpoint IDs do not contain digits")                                       // status code  406
	ErrorMeteringPointAliasTooLong                      error = errors.New("meteringpoint alias too long")                                                  // status code  416
	ErrorWebAccessCodeNot8CharsLong                     error = errors.New("webaccess codes must be 8 characters long")                                     // status code  411
	ErrorWebAccessCodeContainsIllegalChars              error = errors.New("webaccess code contains illegal characters")                                    // status code  406
	ErrorMeteringPointNotFound                          error = errors.New("meteringpoint not found")                                                       // status code  404
	ErrorMeteringPointIsChild                           error = errors.New("meteringpoint can't be child")                                                  // status code  422
	ErrorRelationNotFound                               error = errors.New("relation not found")                                                            // status code  404
	ErrorUnknownError                                   error = errors.New("unknown error")                                                                 // status code  500
	ErrorUnauthorized                                   error = errors.New("unauthorized access")                                                           // status code  401
	ErrorNoValidMeteringPointsInList                    error = errors.New("no meteringpoints in request conforms to valid meteringpoint format")           // status code  400
	ErrorFromDateIsGreaterThanToday                     error = errors.New("requested from date is after today")                                            // status code  400
	ErrorFromDateIsGreaterThanToDate                    error = errors.New("period not allowed, ToDate is before FromDate")                                 // status code  400
	ErrorToDateCanNotBeEqualToFromDate                  error = errors.New("period not allowed, ToDate is equal to FromDate")                               // status code  400
	ErrorToDateIsGreaterThanToday                       error = errors.New("requested to date is after today")                                              // status code  400
	ErrorInvalidDateFormat                              error = errors.New("invalid date format in request")                                                // status code  400
	ErrorInvalidRequestParameters                       error = errors.New("a request parameter is invalid")                                                // status code  400
	ErrorAccessToMeteringPointDenied                    error = errors.New("access to meterpoint denied")                                                   // status code  401
	ErrorNoMeteringPointDataAviliable                   error = errors.New("no meterpoint data aviliable")                                                  // status code  204
	ErrorRequestedAggregationUnavaliable                error = errors.New("requested data aggregation is not supported")                                   // status code  406
	ErrorInvalidMeteringpointId                         error = errors.New("requested meteringpoint ID is not valid")                                       // status code  406
	ErrorDateNotCoveredByAuthorization                  error = errors.New("requested date not covered by Authorization")                                   // status code  401
	ErrorAggrationNotValid                              error = errors.New("requested data aggregation is not supported")                                   // status code  406
	ErrorRequestToHuge                                  error = errors.New("request size too large")                                                        // status code  413
	ErrorInvalidCVR                                     error = errors.New("CVR is invalid")                                                                // status code  403
	ErrorInvalidIncludeFutureMeteringPointsRelatedToCVR error = errors.New("requested future meteringpoints related to CVR are invalid")                    // status code  404
	ErrorInvalidMasterDataFields                        error = errors.New("invalid master data fields")                                                    // status code  417
	ErrorInvalidMeteringPointIds                        error = errors.New("requested meteringpoint IDs are not valid")                                     // status code  406
	ErrorInvalidSignature                               error = errors.New("invalid signature")                                                             // status code  403
	ErrorInvalidSignedByNameId                          error = errors.New("invalid signed by name ID")                                                     // status code  403
	ErrorInvalidSignedDate                              error = errors.New("invalid signed date")                                                           // status code  403
	ErrorInvalidSignedText                              error = errors.New("invalid signed text")                                                           // status code  403
	ErrorInvalidThirdPartyId                            error = errors.New("invalid third party ID. ")                                                      // status code  403
	ErrorInvalidValidFrom                               error = errors.New("invalid from date")                                                             // status code  400
	ErrorInvalidValidTo                                 error = errors.New("invalid to date")                                                               // status code  400
	ErrorValidToBeforeValidFrom                         error = errors.New("requested from date cannot be after requested to date")                         // status code  400
	ErrorValidToOutOfRange                              error = errors.New("requested to date is out of range")                                             // status code  400
	ErrorValidFromOutOfRange                            error = errors.New("requested from date is out of range")                                           // status code  400
	ErrorNoAuthorizationsFound                          error = errors.New("no power of attorneys found")                                                   // status code  400
	ErrorWrongTokenType                                 error = errors.New("request used wrong token type")                                                 // status code  406
	ErrorTokenNotValid                                  error = errors.New("token is invalid")                                                              // status code  401
	ErrorCreatingToken                                  error = errors.New("error creating token")                                                          // status code  500
	ErrorTokenRegistrationFailed                        error = errors.New("token registration failed")                                                     // status code  500
	ErrorTokenAlreadyActive                             error = errors.New("token already active")                                                          // status code  405
	ErrorTokenAlreadyDeactivated                        error = errors.New("token already deactived")                                                       // status code  405
	ErrorTokenMissingTokenId                            error = errors.New("token do not contain a token id")                                               // status code  401
	ErrorThirdPartyNotFound                             error = errors.New("third party not found")                                                         // status code  404
	ErrorThirdPartyWasNotCreated                        error = errors.New("third party not created")                                                       // status code  400
	ErrorThirdPartyAlreadyExist                         error = errors.New("third party already exist")                                                     // status code  208
	ErrorThirdPartyApplictionInPrgress                  error = errors.New("third party application is already in progress")                                // status code  400
	ErrorThirdPartyAlreadyExistButIsInactive            error = errors.New("third party already exist but is inactive")                                     // status code  401
	ErrorThirdPartyAlreadyExistButIsRevoked             error = errors.New("third party already exist but access is revoked")                               // status code  401
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
