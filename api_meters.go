package eloverblik

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type MeteringPoints struct {
	MeteringPointID         string                `json:"meteringPointId"`
	TypeOfMP                string                `json:"typeOfMP"`
	BalanceSupplierName     string                `json:"balanceSupplierName"`
	StreetCode              string                `json:"streetCode"`
	StreetName              string                `json:"streetName"`
	BuildingNumber          string                `json:"buildingNumber"`
	FloorId                 string                `json:"floorId"`
	RoomId                  string                `json:"roomId"`
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
	ConsumerStartDate       string                `json:"consumerStartDate"` // TODO: Parse time
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

type MeteringPointDetails struct {
	Result MeteringPointDetail `json:"result"`
	StatusResponse
}

type MeteringPointDetail struct {
	MeteringPointId                string               `json:"meteringPointId"`
	ParentMeteringPointId          string               `json:"parentMeteringPointId"`
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
	ConsumerStartDate              string               `json:"consumerStartDate"` // TODO: Parse time
	MeterReadingOccurrence         string               `json:"meterReadingOccurrence"`
	MpReadingCharacteristics       string               `json:"mpReadingCharacteristics"`
	MeterCounterDigits             string               `json:"meterCounterDigits"`
	MeterCounterMultiplyFactor     string               `json:"meterCounterMultiplyFactor"`
	MeterCounterUnit               string               `json:"meterCounterUnit"`
	MeterCounterType               string               `json:"meterCounterType"`
	BalanceSupplierName            string               `json:"balanceSupplierName"`
	BalanceSupplierStartDate       string               `json:"balanceSupplierStartDate"` // TODO: Parse time
	TaxReduction                   string               `json:"taxReduction"`
	TaxSettlementDate              string               `json:"taxSettlementDate"` // TODO: Parse time
	MpRelationType                 string               `json:"mpRelationType"`
	StreetCode                     string               `json:"streetCode"`
	StreetName                     string               `json:"streetName"`
	BuildingNumber                 string               `json:"buildingNumber"`
	FloorId                        string               `json:"floorId"`
	RoomId                         string               `json:"roomId"`
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
	FloorId             string `json:"floorId"`
	RoomId              string `json:"roomId"`
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
	MeteringPointId        string `json:"meteringPointId"`
	ParentMeteringPointId  string `json:"parentMeteringPointId"`
	TypeOfMP               string `json:"typeOfMP"`
	MeterReadingOccurrence string `json:"meterReadingOccurrence"`
	MeterNumber            string `json:"meterNumber"`
}

func (c *client) GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetails, error) {

	// Build URL
	_url := c.hostUrl
	_url.Path += "/MeteringPoints/MeteringPoint/GetDetails"

	// Construct body payload
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(meteringPointIDsToRequestStruct(meteringPointIDs))
	if err != nil {
		return nil, err
	}
	// Construct payload and endpoint path
	req, err := http.NewRequest(http.MethodPost, _url.String(), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Make request and parse response
	res, err := c.client.Do(req)
	if isRetryableError(res.StatusCode, err) {
		return c.GetMeteringPointDetails(meteringPointIDs)
	}

	// Decode response result
	var result struct {
		Result []MeteringPointDetails `json:"result"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Result, err
}

func (c *ThirdPartyClient) GetMeteringPoints(scope, identifier string) ([]MeteringPoints, error) {
	return nil, nil
}

func (c *CustomerClient) GetMeteringPoints(includeAll bool) ([]MeteringPoints, error) {

	// Build URL
	_url := c.hostUrl
	_url.RawQuery = url.Values{"includeAll": {strconv.FormatBool(includeAll)}}.Encode()
	_url.Path += "/MeteringPoints/MeteringPoints"

	// Construct payload and endpoint path
	req, err := http.NewRequest(http.MethodGet, _url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	// Make request and parse response
	res, err := c.client.client.Do(req)
	if isRetryableError(res.StatusCode, err) {
		return c.GetMeteringPoints(includeAll)
	}

	// Decode response result
	var result struct {
		Result []MeteringPoints `json:"result"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Result, err
}
