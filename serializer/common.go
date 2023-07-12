package serializer

type Response struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Msg    string      `json:"msg"`
	Error  string      `json:"error"`
}

type DataList struct {
	Items interface{} `json:"items"`
	Total uint        `json:"total"`
}

type TrackedErrorResponse struct {
	Response
	TrackID string `json:"track_id"`
}

func BuildListResponse(item interface{}, total uint) Response {
	return Response{
		Data: DataList{
			Items: item,
			Total: total,
		},
	}
}
