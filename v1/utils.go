package eloverblik

func meteringPointIDsToRequestStruct(IDs []string) meteringPointIDs {
	return meteringPointIDs{MeteringPointID: meteringPointID{MeteringPointIDs: IDs}}
}
