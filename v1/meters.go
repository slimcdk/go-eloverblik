package eloverblik

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type MeteringPoints struct {
	MeteringPointID         string                `json:"meteringPointId"`
	TypeOfMP                string                `json:"typeOfMP"`
	BalanceSupplierName     string                `json:"balanceSupplierName"`
	StreetCode              string                `json:"streetCode"`
	StreetName              string                `json:"streetName"`
	BuildingNumber          string                `json:"buildingNumber"`
	FloorID                 string                `json:"floorId"`
	RoomID                  string                `json:"roomId"`
	Postcode                string                `json:"postcode"`
	CityName                string                `json:"cityName"`
	CitySubDivisionName     string                `json:"citySubDivisionName"`
	MunicipalityCode        string                `json:"municipalityCode"`
	LocationDescription     string                `json:"locationDescription"`
	SettlementMethod        string                `json:"settlementMethod"`
	MeterReadingOccurrence  string                `json:"meterReadingOccurrence"`
	FirstConsumerPartyName  string                `json:"firstConsumerPartyName"`
	SecondConsumerPartyName string                `json:"secondConsumerPartyName"`
	ConsumerCVR             string                `json:"consumerCVR"`
	DataAccessCVR           string                `json:"dataAccessCVR"`
	MeterNumber             string                `json:"meterNumber"`
	ConsumerStartDate       FlexibleTime          `json:"consumerStartDate"`
	HasRelation             bool                  `json:"hasRelation"`
	ChildMeteringPoints     []ChildMeteringPoints `json:"childMeteringPoints"`
}

type ChildMeteringPoints struct {
	MeteringPointID        string `json:"meteringPointId"`
	ParentMeteringPointID  string `json:"parentMeteringPointId"`
	TypeOfMP               string `json:"typeOfMP"`
	MeterReadingOccurrence string `json:"meterReadingOccurrence"`
	MeterNumber            string `json:"meterNumber"`
}

type MeteringPointDetailsResponse struct {
	Result MeteringPointDetail `json:"result,omitempty"`
	StatusResponse
}

type MeteringPointDetail struct {
	MeteringPointID                string               `json:"meteringPointId"`
	ParentMeteringPointID          string               `json:"parentMeteringPointId"`
	TypeOfMP                       string               `json:"typeOfMP"`
	EnergyTimeSeriesMeasureUnit    string               `json:"energyTimeSeriesMeasureUnit"`
	EstimatedAnnualVolume          string               `json:"estimatedAnnualVolume"`
	SettlementMethod               string               `json:"settlementMethod"`
	MeterNumber                    string               `json:"meterNumber"`
	GridOperatorName               string               `json:"gridOperatorName"`
	MeteringGridAreaIdentification string               `json:"meteringGridAreaIdentification"`
	NetSettlementGroup             string               `json:"netSettlementGroup"`
	PhysicalStatusOfMP             string               `json:"physicalStatusOfMP"`
	ConsumerCategory               string               `json:"consumerCategory"`
	PowerLimitKW                   string               `json:"powerLimitKW"`
	PowerLimitA                    string               `json:"powerLimitA"`
	SubTypeOfMP                    string               `json:"subTypeOfMP"`
	ProductionObligation           string               `json:"productionObligation"`
	MpCapacity                     string               `json:"mpCapacity"`
	MpConnectionType               string               `json:"mpConnectionType"`
	DisconnectionType              string               `json:"disconnectionType"`
	Product                        string               `json:"product"`
	ConsumerCVR                    string               `json:"consumerCVR"`
	DataAccessCVR                  string               `json:"dataAccessCVR"`
	ConsumerStartDate              FlexibleTime         `json:"consumerStartDate"`
	MeterReadingOccurrence         string               `json:"meterReadingOccurrence"`
	MpReadingCharacteristics       string               `json:"mpReadingCharacteristics"`
	MeterCounterDigits             string               `json:"meterCounterDigits"`
	MeterCounterMultiplyFactor     string               `json:"meterCounterMultiplyFactor"`
	MeterCounterUnit               string               `json:"meterCounterUnit"`
	MeterCounterType               string               `json:"meterCounterType"`
	BalanceSupplierName            string               `json:"balanceSupplierName"`
	BalanceSupplierStartDate       FlexibleTime         `json:"balanceSupplierStartDate"`
	TaxReduction                   string               `json:"taxReduction"`
	TaxSettlementDate              FlexibleTime         `json:"taxSettlementDate"`
	MpRelationType                 string               `json:"mpRelationType"`
	StreetCode                     string               `json:"streetCode"`
	StreetName                     string               `json:"streetName"`
	BuildingNumber                 string               `json:"buildingNumber"`
	FloorID                        string               `json:"floorId"`
	RoomID                         string               `json:"roomId"`
	Postcode                       string               `json:"postcode"`
	CityName                       string               `json:"cityName"`
	CitySubDivisionName            string               `json:"citySubDivisionName"`
	MunicipalityCode               string               `json:"municipalityCode"`
	LocationDescription            string               `json:"locationDescription"`
	FirstConsumerPartyName         string               `json:"firstConsumerPartyName"`
	SecondConsumerPartyName        string               `json:"secondConsumerPartyName"`
	ContactAddresses               []ContactAddress     `json:"contactAddresses"`
	ChildMeteringPoints            []ChildMeteringPoint `json:"childMeteringPoints"`
}

type ContactAddress struct {
	ContactName1        string `json:"contactName1"`
	ContactName2        string `json:"contactName2"`
	AddressCode         string `json:"addressCode"`
	StreetName          string `json:"streetName"`
	BuildingNumber      string `json:"buildingNumber"`
	FloorID             string `json:"floorId"`
	RoomID              string `json:"roomId"`
	CitySubDivisionName string `json:"citySubDivisionName"`
	Postcode            string `json:"postcode"`
	CityName            string `json:"cityName"`
	CountryName         string `json:"countryName"`
	ContactPhoneNumber  string `json:"contactPhoneNumber"`
	ContactMobileNumber string `json:"contactMobileNumber"`
	ContactEmailAddress string `json:"contactEmailAddress"`
	ContactType         string `json:"contactType"`
}

type ChildMeteringPoint struct {
	MeteringPointID        string `json:"meteringPointId"`
	ParentMeteringPointID  string `json:"parentMeteringPointId"`
	TypeOfMP               string `json:"typeOfMP"`
	MeterReadingOccurrence string `json:"meterReadingOccurrence"`
	MeterNumber            string `json:"meterNumber"`
}

func (c *client) GetMeteringPoints(includeAll bool) ([]MeteringPoints, error) {

	// Ensure access token is fresh
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	// Response struct
	var result struct {
		Result []MeteringPoints `json:"result"`
	}

	// Request preflight
	req := c.resty.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(accessToken).
		SetResult(&result).
		SetPathParams(map[string]string{
			"includeAll": strconv.FormatBool(includeAll),
		})

	// Execute request
	res, err := req.Get("/MeteringPoints/MeteringPoints")
	if err != nil || res.StatusCode() != http.StatusOK {
		return nil, err
	}
	return result.Result, err
}

func (c *client) GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetailsResponse, error) {
	// Ensure access token is fresh
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var path string
	switch c.apiType {
	case CustomerApi:
		path = "/meteringpoints/meteringpoint/getdetails"
	case ThirdPartyApi:
		path = "/meteringpoint/getdetails"
	default:
		return nil, fmt.Errorf("unsupported API type for GetMeteringPointDetails")
	}

	var result struct {
		Result []MeteringPointDetailsResponse `json:"result"`
	}
	var apiErrorMsg string

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetResult(&result).
		SetError(&apiErrorMsg).
		Post(path)

	if err != nil {
		return nil, err
	}
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, nil
}

func (c *client) ExportMasterdata(meteringPointIDs []string) (io.ReadCloser, error) {
	if c.apiType != CustomerApi {
		return nil, fmt.Errorf("ExportMasterdata is only available for Customer API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetDoNotParseResponse(true).
		Post("/meteringpoints/masterdata/export")

	if err != nil || !res.IsSuccess() {
		return nil, fmt.Errorf("failed to export masterdata, status: %s, err: %v", res.Status(), err)
	}

	return res.RawBody(), nil
}
