package eloverblik

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		require.NoError(t, err)
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

		require.Error(t, err)
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

		require.NoError(t, err)
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

	path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/%s", meteringPointID)

	t.Run("successfully deletes relation", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("DELETE", path,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, `{"result": true, "success": true}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		success, err := c.DeleteRelation(meteringPointID)

		require.NoError(t, err)
		assert.True(t, success)
	})

	// The response envelope used to be ignored: success was reported from the HTTP status
	// alone, so a documented business error came back as (false, nil).
	t.Run("handles API error response", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("DELETE", path,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(404, `"[20010] Relation not found"`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		success, err := c.DeleteRelation(meteringPointID)

		require.Error(t, err)
		assert.Equal(t, ErrorRelationNotFound, err)
		assert.False(t, success)
	})

	t.Run("reports a body saying the relation was not deleted", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("DELETE", path,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, `{"result": false, "success": false}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		success, err := c.DeleteRelation(meteringPointID)

		require.NoError(t, err)
		assert.False(t, success)
	})

	t.Run("returns an error on a non-200 status without a message", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("DELETE", path, httpmock.NewStringResponder(404, ""))

		success, err := c.DeleteRelation(meteringPointID)

		require.Error(t, err)
		assert.False(t, success)
	})
}

// TestRetiredRelationEndpoints covers the two relation endpoints Energinet retired with
// DataHub 3.0. Both answer 410 Gone, and the specs only say the body is a string, so every
// shape it can plausibly take must surface as ErrorEndpointRetired. Where the body is a
// JSON string, what the API said must survive too.
func TestRetiredRelationEndpoints(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	calls := []struct {
		name string
		call func() error
	}{
		{"AddRelationByWebAccessCode", func() error {
			_, err := c.AddRelationByWebAccessCode("571313180100000001", "ABCD1234")
			return err
		}},
		{"DeleteRelation", func() error {
			_, err := c.DeleteRelation("571313180100000001")
			return err
		}},
	}

	const message = "Adding metering points with a Web Access Code is no longer supported"
	bodies := []struct {
		name        string
		contentType string
		body        string
		keepsText   bool
	}{
		{"a JSON string", "application/json", `"` + message + `"`, true},
		{"plain text", "text/plain", message, false},
		{"a problem document", "application/problem+json", `{"title":"Gone","status":410,"traceId":"00-abc-def-01"}`, false},
		{"no body", "", "", false},
	}

	for _, call := range calls {
		for _, body := range bodies {
			t.Run(call.name+" answered with "+body.name, func(t *testing.T) {
				httpmock.Reset()
				httpmock.RegisterNoResponder(func(*http.Request) (*http.Response, error) {
					resp := httpmock.NewStringResponse(http.StatusGone, body.body)
					if body.contentType != "" {
						resp.Header.Set("Content-Type", body.contentType)
					}
					return resp, nil
				})

				err := call.call()

				require.ErrorIs(t, err, ErrorEndpointRetired)
				if body.keepsText {
					assert.ErrorContains(t, err, message)
				}
			})
		}
	}
}
