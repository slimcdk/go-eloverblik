package eloverblik

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// testToken builds a JWT carrying the given claims. Only the payload matters: the library
// decodes the claims, it does not verify the signature.
func testToken(t *testing.T, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	assert.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".c2lnbmF0dXJl"
}

func TestParseToken(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	t.Run("third party refresh token", func(t *testing.T) {
		// The refresh token names the role claim "roles".
		claims, err := ParseToken(testToken(t, map[string]any{
			"tokenType":         "THIRDPARTYAPI_Refresh",
			"tokenName":         "christian local testing",
			"tokenid":           "6ad87f99-d536-41a2-9722-622e94822fba",
			"webApp":            "ThirdPartyApp",
			"loginType":         "Certificate",
			"cvr":               "44341603",
			"company":           "Styr paa ApS",
			"userId":            "995172",
			"tpid":              "5c01496f-ca03-4bdc-88b7-114f8277ea42",
			"roles":             "ReadPrivate, ReadBusiness",
			"iss":               "Energinet",
			"aud":               "Energinet",
			"exp":               expiry.Unix(),
			claimGivenName:      "Christian Silas Skjerning",
			claimNameIdentifier: "EIA:c004d233-710c-46e3-a9fa-4d787a9e0052",
		}))

		assert.NoError(t, err)
		assert.Equal(t, "THIRDPARTYAPI_Refresh", claims.TokenType)
		assert.Equal(t, "christian local testing", claims.TokenName)
		assert.Equal(t, "6ad87f99-d536-41a2-9722-622e94822fba", claims.TokenID)
		assert.Equal(t, "Christian Silas Skjerning", claims.Name)
		assert.Equal(t, "EIA:c004d233-710c-46e3-a9fa-4d787a9e0052", claims.Subject)
		assert.Equal(t, "Styr paa ApS", claims.Company)
		assert.Equal(t, "44341603", claims.CVR)
		assert.Equal(t, "995172", claims.UserID)
		assert.Equal(t, "5c01496f-ca03-4bdc-88b7-114f8277ea42", claims.ThirdPartyID)
		assert.Equal(t, []string{"ReadPrivate", "ReadBusiness"}, claims.Roles)
		assert.True(t, claims.ExpiresAt.Equal(expiry))

		assert.True(t, claims.IsRefreshToken())
		assert.False(t, claims.IsDataAccessToken())
		assert.False(t, claims.IsExpired())
		assert.InDelta(t, 24*time.Hour, claims.ExpiresIn(), float64(time.Minute))

		apiType, err := claims.APIType()
		assert.NoError(t, err)
		assert.Equal(t, ThirdPartyApi, apiType)
	})

	t.Run("data access token names the role claim differently", func(t *testing.T) {
		claims, err := ParseToken(testToken(t, map[string]any{
			"tokenType": "ThirdPartyApiDataAccess",
			"exp":       expiry.Unix(),
			claimRole:   "ReadPrivate, ReadBusiness",
		}))

		assert.NoError(t, err)
		assert.Equal(t, []string{"ReadPrivate", "ReadBusiness"}, claims.Roles)
		assert.True(t, claims.IsDataAccessToken())
		assert.False(t, claims.IsRefreshToken())
	})

	t.Run("customer token", func(t *testing.T) {
		claims, err := ParseToken(testToken(t, map[string]any{
			"tokenType": "CUSTOMERAPI_Refresh",
			"exp":       expiry.Unix(),
		}))

		assert.NoError(t, err)

		apiType, err := claims.APIType()
		assert.NoError(t, err)
		assert.Equal(t, CustomerApi, apiType)
	})

	t.Run("expired token", func(t *testing.T) {
		claims, err := ParseToken(testToken(t, map[string]any{
			"tokenType": "CUSTOMERAPI_Refresh",
			"exp":       time.Now().Add(-time.Hour).Unix(),
		}))

		assert.NoError(t, err)
		assert.True(t, claims.IsExpired())
		assert.Zero(t, claims.ExpiresIn())
	})

	t.Run("unknown token type cannot name its API", func(t *testing.T) {
		claims, err := ParseToken(testToken(t, map[string]any{
			"tokenType": "Something_Else",
			"exp":       expiry.Unix(),
		}))

		assert.NoError(t, err)
		_, err = claims.APIType()
		assert.Error(t, err)
	})

	t.Run("rejects malformed tokens", func(t *testing.T) {
		tests := map[string]string{
			"empty":             "",
			"not a jwt":         "just-a-string",
			"too few parts":     "header.payload",
			"payload not b64":   "header.!!!not-base64!!!.signature",
			"payload not json":  "header." + base64.RawURLEncoding.EncodeToString([]byte("not json")) + ".signature",
			"no eloverblik jwt": "header." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"someone"}`)) + ".signature",
		}

		for name, token := range tests {
			t.Run(name, func(t *testing.T) {
				_, err := ParseToken(token)
				assert.Error(t, err)
			})
		}
	})
}

func TestClientTokenClaims(t *testing.T) {
	refreshToken := testToken(t, map[string]any{
		"tokenType": "CUSTOMERAPI_Refresh",
		"tokenName": "my token",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	t.Run("reads the refresh token without a request", func(t *testing.T) {
		c := NewCustomer(refreshToken)

		claims, err := c.RefreshTokenClaims()

		assert.NoError(t, err)
		assert.Equal(t, "my token", claims.TokenName)
		assert.True(t, claims.IsRefreshToken())
	})

	t.Run("a token that is not a JWT is reported", func(t *testing.T) {
		c := NewCustomer("not-a-jwt")

		_, err := c.RefreshTokenClaims()

		assert.Error(t, err)
	})
}
