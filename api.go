package eloverblik

// func (c *client) getRequest(url url.URL, resultStruct interface{}) (int, error) {
// func (c *client) makeRequest(method string, url url.URL, body interface{}, result interface{}) (int, error) {

// 	var buf bytes.Buffer
// 	err := json.NewEncoder(&buf).Encode(body)
// 	if err != nil {
// 		return -1, err
// 	}
// 	// Construct payload and endpoint path
// 	req, err := http.NewRequest(method, url.String(), &buf)
// 	if err != nil {
// 		return -1, err
// 	}
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
// 	if method == http.MethodPost {
// 		req.Header.Set("Content-Type", "application/json")
// 	}

// 	// Make request and parse response
// 	res, err := c.client.Do(req)
// 	if err != nil {
// 		return res.StatusCode, err
// 	}

// 	responseWrapper := ResponseResult{}
// 	json.NewDecoder(res.Body).Decode(&responseWrapper)
// 	result = &responseWrapper.Result
// 	return res.StatusCode, err
// }
