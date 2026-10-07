package eloverblik

func meteringPointIDsToRequestStruct(ids []string) meteringPointIDs {
	return meteringPointIDs{MeteringPointID: meteringPointID{MeteringPointIDs: ids}}
}
