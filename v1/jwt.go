package eloverblik

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Claim keys used by the tokens Eloverblik issues. The refresh token names the role claim
// "roles", while the data access token uses the WS-Federation URI for the same thing.
const (
	claimNameIdentifier = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier"
	claimGivenName      = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"
	claimRole           = "http://schemas.microsoft.com/ws/2008/06/identity/claims/role"
)

// TokenClaims holds the claims carried by an Eloverblik refresh or data access token.
//
// The claims are decoded, not verified: the signing key is Energinet's, so a token can
// only be authenticated by using it. Treat the contents as a description of what the
// token says about itself, useful for telling tokens apart, reading their expiry, or
// seeing which API and roles they were issued for.
type TokenClaims struct {
	// TokenType is the type Eloverblik gives the token, e.g. "THIRDPARTYAPI_Refresh"
	// for a refresh token or "ThirdPartyApiDataAccess" for a data access token.
	TokenType string `json:"tokenType"`
	// TokenName is the name given to the token in the Eloverblik portal.
	TokenName string `json:"tokenName,omitempty"`
	TokenID   string `json:"tokenId,omitempty"`

	// Name is the name of the person the token was issued to.
	Name string `json:"name,omitempty"`
	// Subject identifies the token owner, e.g. "EIA:c004d233-...".
	Subject string `json:"subject,omitempty"`

	Company string `json:"company,omitempty"`
	CVR     string `json:"cvr,omitempty"`
	UserID  string `json:"userId,omitempty"`
	// ThirdPartyID is the third party the token belongs to, on third party tokens.
	ThirdPartyID string `json:"thirdPartyId,omitempty"`

	// Roles are the access roles granted to the token, e.g. ReadPrivate, ReadBusiness.
	Roles     []string `json:"roles,omitempty"`
	LoginType string   `json:"loginType,omitempty"`
	WebApp    string   `json:"webApp,omitempty"`

	Issuer   string `json:"issuer,omitempty"`
	Audience string `json:"audience,omitempty"`

	// ExpiresAt is when the token stops working. A data access token is short lived; a
	// refresh token typically lasts a year.
	ExpiresAt time.Time `json:"expiresAt"`
}

// ParseToken decodes the claims of an Eloverblik token, without verifying its signature.
//
// Example:
//
//	claims, err := eloverblik.ParseToken(refreshToken)
//	fmt.Println(claims.TokenName, claims.ExpiresAt, claims.Roles)
func ParseToken(token string) (TokenClaims, error) {

	var claims TokenClaims

	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return claims, fmt.Errorf("not a JWT: expected 3 dot separated parts, got %d", len(parts))
	}

	// The payload is base64url encoded without padding.
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return claims, fmt.Errorf("could not decode token payload: %w", err)
	}

	// The claim set is flat, but its value types are mixed: exp is a number, everything
	// else a string, and the identity token is a blob we deliberately do not expose.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return claims, fmt.Errorf("could not parse token payload: %w", err)
	}

	claims.TokenType = rawString(raw, "tokenType")
	claims.TokenName = rawString(raw, "tokenName")
	claims.TokenID = rawString(raw, "tokenid", "jti")
	claims.Name = rawString(raw, claimGivenName)
	claims.Subject = rawString(raw, claimNameIdentifier, "sub")
	claims.Company = rawString(raw, "company")
	claims.CVR = rawString(raw, "cvr")
	claims.UserID = rawString(raw, "userId")
	claims.ThirdPartyID = rawString(raw, "tpid")
	claims.LoginType = rawString(raw, "loginType")
	claims.WebApp = rawString(raw, "webApp")
	claims.Issuer = rawString(raw, "iss")
	claims.Audience = rawString(raw, "aud")

	// "ReadPrivate, ReadBusiness" under either of the two role claim names.
	for _, role := range strings.Split(rawString(raw, "roles", claimRole), ",") {
		if role = strings.TrimSpace(role); role != "" {
			claims.Roles = append(claims.Roles, role)
		}
	}

	if expiry, ok := raw["exp"]; ok {
		var seconds int64
		if err := json.Unmarshal(expiry, &seconds); err != nil {
			return claims, fmt.Errorf("could not parse token expiry: %w", err)
		}
		claims.ExpiresAt = time.Unix(seconds, 0).In(cph)
	}

	if claims.TokenType == "" && claims.ExpiresAt.IsZero() {
		return claims, fmt.Errorf("token carries no Eloverblik claims")
	}

	return claims, nil
}

// rawString returns the first of the given claims that holds a string.
func rawString(raw map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err == nil && text != "" {
			return text
		}
	}
	return ""
}

// IsExpired reports whether the token has expired.
func (tc TokenClaims) IsExpired() bool {
	return !tc.ExpiresAt.IsZero() && !time.Now().Before(tc.ExpiresAt)
}

// ExpiresIn reports how long the token remains valid. It is zero once expired.
func (tc TokenClaims) ExpiresIn() time.Duration {
	if tc.ExpiresAt.IsZero() || tc.IsExpired() {
		return 0
	}
	return time.Until(tc.ExpiresAt)
}

// IsRefreshToken reports whether the token is a refresh token, i.e. the long lived one
// generated in the Eloverblik portal and handed to NewCustomer or NewThirdParty.
func (tc TokenClaims) IsRefreshToken() bool {
	return strings.Contains(strings.ToLower(tc.TokenType), "refresh")
}

// IsDataAccessToken reports whether the token is a short lived data access token, i.e.
// the one the client fetches from /token and sends on every other request.
func (tc TokenClaims) IsDataAccessToken() bool {
	return strings.Contains(strings.ToLower(tc.TokenType), "dataaccess")
}

// APIType reports which API the token was issued for.
func (tc TokenClaims) APIType() (apiType, error) {
	switch {
	case strings.Contains(strings.ToLower(tc.TokenType), "thirdparty"):
		return ThirdPartyApi, nil
	case strings.Contains(strings.ToLower(tc.TokenType), "customer"):
		return CustomerApi, nil
	default:
		return CustomerApi, fmt.Errorf("cannot tell the API from token type '%s'", tc.TokenType)
	}
}

// RefreshTokenClaims decodes the claims of the refresh token the client was created with.
// It performs no request.
func (c *client) RefreshTokenClaims() (TokenClaims, error) {
	return ParseToken(c.refreshToken)
}

// DataAccessTokenClaims decodes the claims of the client's data access token, fetching
// one first if the client does not hold one yet.
func (c *client) DataAccessTokenClaims() (TokenClaims, error) {
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return TokenClaims{}, err
	}
	return ParseToken(accessToken)
}
