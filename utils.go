package eloverblik

func isAny(check interface{}, against ...interface{}) bool {
	for _, a := range against {
		if check == a {
			return true
		}
	}
	return false
}

func validAggregation(aggregation Aggregation) bool {
	return isAny(
		aggregation,
		AggregationActual,
		AggregationQuarter,
		AggregationHour,
		AggregationDay,
		AggregationMonth,
		AggregationYear,
	)
}

func meteringPointIDsToRequestStruct(IDs []string) meteringPointIDs {
	return meteringPointIDs{MeteringPointID: meteringPointID{MeteringPointIDs: IDs}}
}
