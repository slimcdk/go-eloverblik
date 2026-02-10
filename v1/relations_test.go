package eloverblik

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestAddRelationByID(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	meteringPointIDs := []string{"571313180100000001"}

	t.Run("successfully adds relation by ID", func(t *testing.T) {
		httpmock.Reset()
		mockResponse := `{
			"result": [
				{ "success": true, "id": "571313180100000001", "result": "Relation created" }
			]
		}`
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/relation/add",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		responses, err := c.AddRelationByID(meteringPointIDs)

		assert.NoError(t, err)
		if assert.Len(t, responses, 1) {
			assert.True(t, responses[0].Success)
			assert.Equal(t, "Relation created", responses[0].Result)
		}
	})

	t.Run("handles API error response", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/relation/add",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(404, `"[20000] Invalid metering point ID"`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		_, err := c.AddRelationByID(meteringPointIDs)

		assert.Error(t, err)
		assert.Equal(t, ErrorWrongMeteringPointIdOrWebAccessCode, err)
	})
}

func TestAddRelationByWebAccessCode(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	meteringPointID := "571313180100000001"
	webAccessCode := "12345678"

	t.Run("successfully adds relation by web access code", func(t *testing.T) {
		httpmock.Reset()
		mockResponse := `{ "result": "Relation created" }`
		path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/add/%s/%s", meteringPointID, webAccessCode)
		httpmock.RegisterResponder("PUT", path,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.AddRelationByWebAccessCode(meteringPointID, webAccessCode)

		assert.NoError(t, err)
		assert.Equal(t, "Relation created", result)
	})
}

func TestDeleteRelation(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	meteringPointID := "571313180100000001"

	t.Run("successfully deletes relation", func(t *testing.T) {
		httpmock.Reset()
		path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/%s", meteringPointID)
		httpmock.RegisterResponder("DELETE", path, httpmock.NewStringResponder(200, `{"result": true}`))

		success, err := c.DeleteRelation(meteringPointID)

		assert.NoError(t, err)
		assert.True(t, success)
	})

	t.Run("returns false on non-200 status", func(t *testing.T) {
		httpmock.Reset()
		path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/%s", meteringPointID)
		httpmock.RegisterResponder("DELETE", path, httpmock.NewStringResponder(404, ""))

		success, err := c.DeleteRelation(meteringPointID)

		assert.NoError(t, err)
		assert.False(t, success)
	})
}
