package eloverblik

type Aggregation string

const (
	HTTP_SCHEME string = "https"
	TEST_HOST   string = "apipreprod.eloverblik.dk"
	PROD_HOST   string = "api.eloverblik.dk"

	CUSTOMER_ENDPOINTS  string = "CustomerApi"
	THIRDPART_ENDPOINTS string = "ThirdPartyApi"

	API_VERSION_1 string = "api"
)

const (
	AggregationActual  Aggregation = "Actual"
	AggregationQuarter Aggregation = "Quarter"
	AggregationHour    Aggregation = "Hour"
	AggregationDay     Aggregation = "Day"
	AggregationMonth   Aggregation = "Month"
	AggregationYear    Aggregation = "Year"
)

const (
	DateFormat string = "2006-01-02"
)
