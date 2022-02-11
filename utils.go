package eloverblik

func isAny(check interface{}, against ...interface{}) bool {
	for _, a := range against {
		if check == a {
			return true
		}
	}
	return false
}

func verifyAggregation(aggregation Aggregation) (Aggregation, error) {
	if !isAny(aggregation, AggregationActual, AggregationQuarter, AggregationHour, AggregationDay, AggregationMonth, AggregationYear) {
		return aggregation, ErrorAggrationNotValid
	}
	return aggregation, nil
}

func meteringPointIDsToRequestStruct(IDs []string) meteringPointIDs {
	return meteringPointIDs{MeteringPointID: meteringPointID{MeteringPointIDs: IDs}}
}
