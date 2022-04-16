package eloverblik

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func isAny(check interface{}, against ...interface{}) bool {
	for _, a := range against {
		if check == a {
			return true
		}
	}
	return false
}

func validAggregation(aggregation Aggregation) bool {
	return isAny(aggregation, Actual, Quarter, Hour, Day, Month, Year)
}

func meteringPointIDsToRequestStruct(IDs []string) meteringPointIDs {
	return meteringPointIDs{MeteringPointID: meteringPointID{MeteringPointIDs: IDs}}
}

func prettyPrint(emp ...interface{}) {
	empJSON, err := json.MarshalIndent(emp, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println(string(empJSON))
}

func timeRangeVoilation(from, to time.Time) bool {
	return time.Duration(to.Sub(from).Hours()) > MaximumRequestDuration
}
