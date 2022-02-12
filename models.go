package eloverblik

type meteringPointIDs struct {
	MeteringPointID meteringPointID `json:"meteringPoints"`
}

type meteringPointID struct {
	MeteringPointIDs []string `json:"meteringPoint"`
}

type StatusResponse struct {
	Success    bool   `json:"success"`
	ErrorCode  string `json:"errorCode"` // TODO: Could perhaps be an integer
	ErrorText  string `json:"errorText"`
	ID         string `json:"id"`
	StackTrace string `json:"stackTrace"`
}
