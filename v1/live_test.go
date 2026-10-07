//go:build live

package eloverblik

// The live tests check the client against the production API at api.eloverblik.dk, with
// the refresh tokens of the person running them. They verify the client by hand, e.g.
// after Energinet changes the API, and never run in CI or in a plain go test: only the
// live build tag compiles them in, and each test skips unless its token is set.
//
//	export ELO_CUSTOMER_TOKEN='...'    # a Customer API refresh token, and/or
//	export ELO_THIRDPARTY_TOKEN='...'  # a Third-Party API refresh token
//	go test -tags live -count=1 -v -run Live ./v1/
//
// They check the shape of every answer, never its values: no call fails, metering point
// IDs and GLNs have their format, the fields of an address are filled in, a week of time
// series is not all zeros, and no response carries a field the client would drop. Those
// fail the test. A field the client decodes that a response leaves out is only logged,
// because the API leaves out empty fields. They read only metering points that should
// have data for last week: the customer test skips those the user has moved out of, the
// third-party test those whose access does not cover the week, and both read time series
// only for connected metering points. They only read: no relation is added or deleted.
//
// They log counts, codes, resolutions and field names, never a value from a response.
// Errors go through describeShapeError, which leaves out the values, URLs and IDs an error
// can quote, and resty's own log is silenced, so the output holds no customer data and can
// be shared.

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-resty/resty/v2"
)

const (
	liveCustomerTokenEnv   = "ELO_CUSTOMER_TOKEN"
	liveThirdPartyTokenEnv = "ELO_THIRDPARTY_TOKEN"

	// liveMaxIDs is the number of metering points Energinet recommends per request.
	liveMaxIDs = 10

	// liveMaxAuthorizations caps the authorizations the third-party test reads metering
	// points from, to keep the number of requests small.
	liveMaxAuthorizations = 3

	// liveConnected is physicalStatusOfMP of a connected metering point.
	liveConnected = "E22"
)

var (
	liveTypeOfMP      = regexp.MustCompile(`^[A-Z]\d{2}$`)
	livePostcode      = regexp.MustCompile(`^\d{4}$`)
	liveThreeDigits   = regexp.MustCompile(`^\d{3}$`)
	liveCVR           = regexp.MustCompile(`^\d{8}$`)
	liveUUID          = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	liveEIC           = regexp.MustCompile(`^[0-9]{2}[A-Z][0-9A-Z-]{12}[0-9A-Z]$`)
	liveColumnName    = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _.,:()/%&'+-]*$`)
	liveFourDigits    = regexp.MustCompile(`\d{4}`)
	liveExportColumns = []string{"MålepunktsID", "Fra_dato", "Til_dato", "Mængde", "Måleenhed", "Kvalitet", "Type", "Målepunktstype_Kode", "Målepunktstype"}
)

// The columns of the master data and charges exports, as llms.md lists them.
var (
	liveMasterdataColumns = []string{
		"MålepunktsID", "MålepunktsID_hovedmåler", "Alias", "Målepunktstype_Kode", "Målepunktstype",
		"Netområde", "Nettoafregningsgruppe", "Tilslutningsstatus_Kode", "Tilslutningsstatus",
		"Branchekode", "Effektgrænse_kW", "Effektgrænse_ampere", "Målepunktsart_Kode",
		"Målepunktsart", "Aftagepligt_Kode", "Aftagepligt", "Anlægskapacitet",
		"Tilslutningstype_Kode", "Tilslutningstype", "Afbrydelsesart_Kode", "Afbrydelsesart",
		"Produkt_Kode", "Produkt", "Måleenhed", "Adressekode", "Vejnavn", "Husnummer", "Etage",
		"Dørnummer", "Postnummer", "By", "Stednavn", "Kommunekode", "Målepunktskommentar",
		"Kundenavn", "Kundenavn_2", "CVR-nummer", "DataadgangsCVR-nummer", "Afregningsform_Kode",
		"Afregningsform", "Elleverandør", "Elleverandørstartdato", "Kunde_start_dato",
		"Aflæsningsfrekvens_Kode", "Aflæsningsfrekvens", "Anslået_årsforbrug",
		"Aflæsningsmåde_Kode", "Aflæsningsmåde", "Målernummer", "Målercifre",
		"Måleromregningsfaktor", "Målerenhed", "Målertype_Kode", "Målertype",
		"Reduceret_elafgift_Kode", "Reduceret_elafgift", "Elvarmestartdato", "Netvirksomhed",
		"Teknisk_kontakt_Navn", "Teknisk_kontakt_Navn2", "Teknisk_kontakt_Vejnavn",
		"Teknisk_kontakt_Husnr.", "Teknisk_kontakt_Etage", "Teknisk_kontakt_Dør",
		"Teknisk_kontakt_Postnr", "Teknisk_kontakt_By", "Teknisk_kontakt_Stednavn",
		"Teknisk_kontakt_Land", "Teknisk_kontakt_Telefonnr.", "Teknisk_kontakt_Mobilnr.",
		"Teknisk_kontakt_E-mail", "Teknisk_kontakt_Attention", "Teknisk_kontakt_Postbox",
		"Teknisk_kontakt_beskyttet_adresse_Kode", "Teknisk_kontakt_beskyttet_adresse",
		"Juridisk_kontakt_Navn", "Juridisk_kontakt_Navn2", "Juridisk_kontakt_Vejnavn",
		"Juridisk_kontakt_Husnr.", "Juridisk_kontakt_Etage", "Juridisk_kontakt_Dør",
		"Juridisk_kontakt_Postnr.", "Juridisk_kontakt_By", "Juridisk_kontakt_Stednavn",
		"Juridisk_kontakt_Land", "Juridisk_kontakt_Telefonnr.", "Juridisk_kontakt_Mobilnr.",
		"Juridisk_kontakt_E-mail", "Juridisk_kontakt_Attention", "Juridisk_kontakt_Postbox",
		"Juridisk_kontakt_beskyttet_adresse_Kode", "Juridisk_kontakt_beskyttet_adresse",
		"DAR_adresse_konflikt_Kode", "DAR_adresse_konflikt", "DAR_reference", "Beskyttet_Navn_Kode",
		"Beskyttet_Navn", "Energi_type_Kode", "Energi_type", "Elleverandør_Id",
		"Elleverandør_Id_type", "Netvirksomhed_Id", "Netvirksomhed_Id_type",
	}
	liveChargesColumns = []string{
		"MålepunktsID", "Pristype", "Pris_ID", "Navn", "Beskrivelse", "Ejer", "Gyldig_fra",
		"Gyldig_til", "Position", "Pris (Ekskl. Moms)", "Mængde",
	}
)

// liveRecorder keeps the body of the last response, for the field checks. The /token
// response is never kept.
type liveRecorder struct {
	mu   sync.Mutex
	body []byte
}

// take returns the body of the last response and forgets it, so a call that records
// nothing cannot be checked against the body of the call before it.
func (r *liveRecorder) take() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	body := r.body
	r.body = nil
	return body
}

// liveQuietLogger silences resty, which would otherwise write each failed attempt's error
// to stderr as it is. The tests report every error themselves, through describeShapeError.
type liveQuietLogger struct{}

func (liveQuietLogger) Errorf(string, ...any) {}
func (liveQuietLogger) Warnf(string, ...any)  {}
func (liveQuietLogger) Debugf(string, ...any) {}

// liveClient builds a client from the refresh token in env, and skips the test when env
// is not set: without a token, a live test sends nothing.
func liveClient[T any](t *testing.T, env string, newClient func(string, ...Option) T) *client {
	t.Helper()

	token := os.Getenv(env)
	if token == "" {
		t.Skipf("%s is not set", env)
	}

	c, ok := any(newClient(token)).(*client)
	if !ok {
		t.Fatalf("%s: the constructor did not return a *client", env)
	}
	c.resty.SetLogger(liveQuietLogger{})
	return c
}

// recordResponses makes the client keep the body of every response but the /token one in
// the returned recorder.
func recordResponses(c *client) *liveRecorder {
	rec := &liveRecorder{}
	c.resty.OnAfterResponse(func(_ *resty.Client, res *resty.Response) error {
		raw := res.RawResponse
		if raw == nil || raw.Request == nil || strings.HasSuffix(raw.Request.URL.Path, "/token") {
			return nil
		}
		rec.mu.Lock()
		rec.body = res.Body()
		rec.mu.Unlock()
		return nil
	})
	return rec
}

// liveOK reports a failed call and returns false, so the checks of its result are skipped.
func liveOK(t *testing.T, what string, err error) bool {
	t.Helper()
	if err != nil {
		t.Errorf("%s: %s", what, describeShapeError(err))
		return false
	}
	return true
}

// liveItemOK reports a metering point that failed on its own inside a successful response.
func liveItemOK(t *testing.T, what string, status StatusResponse) bool {
	t.Helper()
	if err := status.Err(); err != nil {
		t.Errorf("%s: %s", what, describeShapeError(err))
		return false
	}
	return true
}

// liveFields checks the last response against T, the type the client decodes it into:
// a key T has no field for is an error, because the client drops it; a field of T no
// object carried is logged, because the API may leave out an empty field.
func liveFields[T any](t *testing.T, rec *liveRecorder, what string) {
	t.Helper()

	body := rec.take()
	if body == nil {
		t.Errorf("%s: no response body was recorded", what)
		return
	}

	unknown, missing, err := jsonFieldDrift(body, reflect.TypeFor[T]())
	if err != nil {
		t.Errorf("%s: the response is not JSON: %s", what, describeShapeError(err))
		return
	}
	if len(unknown) > 0 {
		t.Errorf("%s: the API sends fields the client does not decode: %s", what, strings.Join(unknown, ", "))
	}
	if len(missing) > 0 {
		t.Logf("%s: fields the client decodes that this response left out: %s", what, strings.Join(missing, ", "))
	}
}

// liveSameIDs reports requested metering points the answer leaves out, and metering points
// it holds that were not requested. A metering point that comes back once per access
// period is not an error.
func liveSameIDs(t *testing.T, what string, requested, answered []string) {
	t.Helper()
	if missing := liveCountNotIn(requested, answered); missing > 0 {
		t.Errorf("%s: %d of the %d requested metering points are not in the answer", what, missing, len(requested))
	}
	if extra := liveCountNotIn(answered, requested); extra > 0 {
		t.Errorf("%s: %d metering points in the answer were not requested", what, extra)
	}
}

// liveCountNotIn counts the distinct values of a that b does not hold.
func liveCountNotIn(a, b []string) int {
	count := 0
	seen := map[string]bool{}
	for _, v := range a {
		if !seen[v] && !slices.Contains(b, v) {
			count++
		}
		seen[v] = true
	}
	return count
}

// liveFilled reports each named value that is empty.
func liveFilled(t *testing.T, what string, values map[string]string) {
	t.Helper()
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if strings.TrimSpace(values[name]) == "" {
			t.Errorf("%s: %s is empty", what, name)
		}
	}
}

// liveMatches reports a value that is set but does not have its format. An empty value
// passes: liveFilled checks the values that must be set.
func liveMatches(t *testing.T, what, name, value string, format *regexp.Regexp) {
	t.Helper()
	if value != "" && !format.MatchString(value) {
		t.Errorf("%s: %s does not have its format %s", what, name, format)
	}
}

// liveAddress holds the fields an address of a metering point needs. A metering point
// without a house number, such as street lighting, has a location description instead.
type liveAddress struct {
	streetName, buildingNumber, postcode, cityName, municipalityCode, locationDescription string
}

func (a liveAddress) check(t *testing.T, what string) {
	t.Helper()
	liveFilled(t, what, map[string]string{
		"streetName":       a.streetName,
		"postcode":         a.postcode,
		"cityName":         a.cityName,
		"municipalityCode": a.municipalityCode,
	})
	if strings.TrimSpace(a.buildingNumber) == "" && strings.TrimSpace(a.locationDescription) == "" {
		t.Errorf("%s: neither buildingNumber nor locationDescription is set", what)
	}
	liveMatches(t, what, "postcode", a.postcode, livePostcode)
	liveMatches(t, what, "municipalityCode", a.municipalityCode, liveThreeDigits)
}

// liveGSRN reports a metering point ID that is not an 18-digit GSRN with a valid check digit.
func liveGSRN(t *testing.T, what, name, id string) {
	t.Helper()
	if !validGSRN(id) {
		t.Errorf("%s: %s is not a metering point ID (18 digits starting with 57, valid check digit)", what, name)
	}
}

// liveParty reports a market participant ID that is set but has not the format of its
// scheme: a GLN (13 digits, valid check digit) or an EIC (16 characters). With no scheme
// given, either passes.
func liveParty(t *testing.T, what, name, id, scheme string) {
	t.Helper()
	if id == "" {
		return
	}
	gln, eic := validGS1(id, 13), liveEIC.MatchString(id)
	switch scheme {
	case "GLN":
		if !gln {
			t.Errorf("%s: %s is not a GLN (13 digits, valid check digit)", what, name)
		}
	case "EIC":
		if !eic {
			t.Errorf("%s: %s is not an EIC (16 characters)", what, name)
		}
	case "":
		if !gln && !eic {
			t.Errorf("%s: %s is neither a GLN nor an EIC", what, name)
		}
	default:
		t.Errorf("%s: the scheme of %s is neither GLN nor EIC", what, name)
	}
}

// liveDate parses the dates the API sends as plain strings: a date, or a timestamp with
// or without a UTC offset.
func liveDate(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", time.DateOnly} {
		if d, err := time.ParseInLocation(layout, s, cph); err == nil {
			return d, true
		}
	}
	return time.Time{}, false
}

// liveShape describes the form of a value without the value: its length, how many of its
// characters are digits and letters, and which other characters it holds.
func liveShape(v string) string {
	var digits, letters int
	var other []rune
	for _, r := range v {
		switch {
		case unicode.IsDigit(r):
			digits++
		case unicode.IsLetter(r):
			letters++
		case !slices.Contains(other, r):
			other = append(other, r)
		}
	}
	return fmt.Sprintf("%d characters: %d digits, %d letters, other characters %q", utf8.RuneCountInString(v), digits, letters, string(other))
}

// liveChildren checks the child metering points of a parent.
func liveChildren(t *testing.T, what, parentID string, children []ChildMeteringPoint) {
	t.Helper()
	for j, child := range children {
		cw := fmt.Sprintf("%s: child %d", what, j)
		liveGSRN(t, cw, "meteringPointId", child.MeteringPointID)
		if child.ParentMeteringPointID != parentID {
			t.Errorf("%s: parentMeteringPointId is not the ID of its parent", cw)
		}
	}
}

func TestLiveCustomer(t *testing.T) {
	c := liveClient(t, liveCustomerTokenEnv, NewCustomer)
	liveCustomer(t, c, recordResponses(c))
}

func TestLiveThirdParty(t *testing.T) {
	c := liveClient(t, liveThirdPartyTokenEnv, NewThirdParty)
	liveThirdParty(t, c, recordResponses(c))
}

func liveCustomer(t *testing.T, c *client, rec *liveRecorder) {
	if !liveAliveAndTokens(t, c, CustomerApi) {
		t.Fatal("no data access token: stopping, so no further /token request is sent")
	}

	var ids []string
	movedOut := 0
	t.Run("metering points", func(t *testing.T) {
		for _, includeAll := range []bool{false, true} {
			what := fmt.Sprintf("GetMeteringPoints(includeAll=%t)", includeAll)

			mps, err := c.GetMeteringPoints(includeAll)
			if !liveOK(t, what, err) {
				continue
			}
			liveFields[struct {
				Result []MeteringPoints `json:"result"`
			}](t, rec, what)

			t.Logf("%s: %d metering points", what, len(mps))
			if len(mps) == 0 {
				t.Errorf("%s: no metering points", what)
			}

			for i, mp := range mps {
				mw := fmt.Sprintf("%s: metering point %d", what, i)
				liveGSRN(t, mw, "meteringPointId", mp.MeteringPointID)
				liveFilled(t, mw, map[string]string{"typeOfMP": mp.TypeOfMP})
				liveMatches(t, mw, "typeOfMP", mp.TypeOfMP, liveTypeOfMP)
				liveMatches(t, mw, "consumerCVR", mp.ConsumerCVR, liveCVR)
				liveMatches(t, mw, "dataAccessCVR", mp.DataAccessCVR, liveCVR)
				liveAddress{mp.StreetName, mp.BuildingNumber, mp.Postcode, mp.CityName, mp.MunicipalityCode, mp.LocationDescription}.check(t, mw)
				for j, child := range mp.ChildMeteringPoints {
					cw := fmt.Sprintf("%s: child %d", mw, j)
					liveGSRN(t, cw, "meteringPointId", child.MeteringPointID)
					if child.ParentMeteringPointID != mp.MeteringPointID {
						t.Errorf("%s: parentMeteringPointId is not the ID of its parent", cw)
					}
				}

				if includeAll || !mp.HasRelation {
					continue
				}
				if mp.IsMovedOut {
					movedOut++
					continue
				}
				if len(ids) < liveMaxIDs {
					ids = append(ids, mp.MeteringPointID)
				}
			}
		}
	})
	if movedOut > 0 {
		t.Logf("skipping %d metering points the user has moved out of", movedOut)
	}
	if len(ids) == 0 {
		t.Skip("no related metering point the user still lives at, so nothing to read further")
	}
	t.Logf("reading %d metering points further", len(ids))

	from, to := liveLastWeek(t)
	connected := liveMeteringPointReads(t, c, rec, ids, from, to)

	t.Run("exports", func(t *testing.T) {
		liveExportTimeSeries(t, c, connected, from, to)
		liveExport(t, "ExportMasterdata", liveMasterdataColumns, func() (io.ReadCloser, error) { return c.ExportMasterdata(ids) })
		liveExport(t, "ExportCharges", liveChargesColumns, func() (io.ReadCloser, error) { return c.ExportCharges(ids) })
	})
}

func liveThirdParty(t *testing.T, c *client, rec *liveRecorder) {
	if !liveAliveAndTokens(t, c, ThirdPartyApi) {
		t.Fatal("no data access token: stopping, so no further /token request is sent")
	}

	from, to := liveLastWeek(t)

	var ids []string
	t.Run("authorizations", func(t *testing.T) {
		what := "GetAuthorizations"
		auths, err := c.GetAuthorizations()
		if !liveOK(t, what, err) {
			return
		}
		liveFields[struct {
			Result []Authorization `json:"result"`
		}](t, rec, what)

		t.Logf("%s: %d authorizations", what, len(auths))
		if len(auths) == 0 {
			t.Errorf("%s: no authorizations", what)
		}

		read, notCovering := 0, 0
		for i, auth := range auths {
			aw := fmt.Sprintf("%s: authorization %d", what, i)
			liveFilled(t, aw, map[string]string{
				"id":             auth.ID,
				"thirdPartyName": auth.ThirdPartyName,
				"validFrom":      auth.ValidFrom,
				"validTo":        auth.ValidTo,
			})
			liveMatches(t, aw, "customerCVR", auth.CustomerCVR, liveCVR)
			if auth.Timestamp.IsZero() {
				t.Errorf("%s: timeStamp is empty", aw)
			}
			validFrom, okFrom := liveDate(auth.ValidFrom)
			validTo, okTo := liveDate(auth.ValidTo)
			if !okFrom || !okTo {
				t.Errorf("%s: validFrom or validTo is not a date", aw)
				continue
			}

			// Only an authorization that covers last week can be read without per item errors
			if validFrom.After(from) || validTo.Before(to) {
				notCovering++
				continue
			}
			if read >= liveMaxAuthorizations || len(ids) >= liveMaxIDs {
				continue
			}
			read++
			ids = liveAuthorizationMeteringPoints(t, c, rec, aw, auth.ID, ids, from, to)
		}
		if notCovering > 0 {
			t.Logf("%s: skipping %d authorizations that do not cover last week", what, notCovering)
		}
	})
	if len(ids) == 0 {
		t.Skip("no metering point with access over last week, so nothing to read further")
	}
	t.Logf("reading %d metering points further", len(ids))

	liveMeteringPointReads(t, c, rec, ids, from, to)
}

// liveAuthorizationMeteringPoints reads the metering points of one authorization and adds
// to ids, up to liveMaxIDs, those the third party has access to over [from, to).
func liveAuthorizationMeteringPoints(t *testing.T, c *client, rec *liveRecorder, what, authID string, ids []string, from, to time.Time) []string {
	t.Helper()

	idsWhat := what + ": GetMeteringPointIDsForScope"
	mpIDs, err := c.GetMeteringPointIDsForScope(AuthScopeID, authID)
	idsOK := liveOK(t, idsWhat, err)
	if idsOK {
		liveFields[struct {
			Result []string `json:"result"`
		}](t, rec, idsWhat)
		t.Logf("%s: %d metering point IDs", idsWhat, len(mpIDs))
		for i, id := range mpIDs {
			liveGSRN(t, idsWhat, fmt.Sprintf("ID %d", i), id)
		}
	}

	mpsWhat := what + ": GetMeteringPointsForScope"
	mps, err := c.GetMeteringPointsForScope(AuthScopeID, authID)
	if !liveOK(t, mpsWhat, err) {
		return ids
	}
	liveFields[struct {
		Result []ThirdPartyMeteringPoint `json:"result"`
	}](t, rec, mpsWhat)
	t.Logf("%s: %d metering points", mpsWhat, len(mps))

	var mpsIDs, childIDs []string
	noAccess := 0
	for i, mp := range mps {
		mpsIDs = append(mpsIDs, mp.MeteringPointID)
		for _, child := range mp.ChildMeteringPoints {
			childIDs = append(childIDs, child.MeteringPointID)
		}

		mw := fmt.Sprintf("%s: metering point %d", mpsWhat, i)
		liveGSRN(t, mw, "meteringPointId", mp.MeteringPointID)
		liveFilled(t, mw, map[string]string{"typeOfMP": mp.TypeOfMP, "accessFrom": mp.AccessFrom})
		liveMatches(t, mw, "typeOfMP", mp.TypeOfMP, liveTypeOfMP)
		liveMatches(t, mw, "consumerCVR", mp.ConsumerCVR, liveCVR)
		liveMatches(t, mw, "dataAccessCVR", mp.DataAccessCVR, liveCVR)
		liveAddress{mp.StreetName, mp.BuildingNumber, mp.Postcode, mp.CityName, mp.MunicipalityCode, mp.LocationDescription}.check(t, mw)
		liveChildren(t, mw, mp.MeteringPointID, mp.ChildMeteringPoints)

		accessFrom, okFrom := liveDate(mp.AccessFrom)
		accessTo, okTo := liveDate(mp.AccessTo)
		if mp.AccessFrom != "" && !okFrom {
			t.Errorf("%s: accessFrom is not a date", mw)
		}
		if mp.AccessTo != "" && !okTo {
			t.Errorf("%s: accessTo is not a date", mw)
		}

		// Only a metering point the third party can read all of last week of has data for it
		if !okFrom || accessFrom.After(from) || (mp.AccessTo != "" && (!okTo || accessTo.Before(to))) {
			noAccess++
			continue
		}
		if len(ids) < liveMaxIDs && !slices.Contains(ids, mp.MeteringPointID) {
			ids = append(ids, mp.MeteringPointID)
		}
	}
	if noAccess > 0 {
		t.Logf("%s: skipping %d metering points whose access does not cover last week", mpsWhat, noAccess)
	}
	if idsOK {
		liveScopeIDs(t, mpsWhat, mpIDs, mpsIDs, childIDs)
	}

	return ids
}

// liveScopeIDs compares the IDs GetMeteringPointIDsForScope lists with the metering points
// GetMeteringPointsForScope returns for the same authorization. The OpenAPI documents do
// not say the two lists match, so a difference is logged, not an error. An ID can name a
// child metering point, which GetMeteringPointsForScope nests under its parent.
func liveScopeIDs(t *testing.T, what string, listed, returned, children []string) {
	t.Helper()

	var asChild, neither int
	for _, id := range slices.Compact(slices.Sorted(slices.Values(listed))) {
		switch {
		case slices.Contains(returned, id):
		case slices.Contains(children, id):
			asChild++
		default:
			neither++
		}
	}
	if asChild > 0 {
		t.Logf("%s: %d of the IDs GetMeteringPointIDsForScope lists are child metering points, returned under their parent", what, asChild)
	}
	if neither > 0 {
		t.Logf("%s: %d of the IDs GetMeteringPointIDsForScope lists are not returned, neither as a metering point nor as a child", what, neither)
	}
	if extra := liveCountNotIn(returned, listed); extra > 0 {
		t.Logf("%s: %d metering points that GetMeteringPointIDsForScope does not list", what, extra)
	}
}

// liveLastWeek returns last week, Monday to Monday in Copenhagen time: whole days after
// DataHub 3.0 went live, which every connected metering point has data for.
func liveLastWeek(t *testing.T) (from, to time.Time) {
	t.Helper()
	from, to, err := GetDatesFromPeriod(LastWeek)
	if err != nil {
		t.Fatalf("GetDatesFromPeriod(LastWeek): %v", err)
	}
	return from, to
}

// liveAliveAndTokens checks that the API is up and that both tokens are what they claim.
// It returns false when no data access token could be fetched: every further call would
// then send /token again, and the API allows 2 /token calls a minute.
func liveAliveAndTokens(t *testing.T, c *client, want apiType) bool {
	t.Run("alive", func(t *testing.T) {
		alive, err := c.IsAlive()
		if liveOK(t, "IsAlive", err) && !alive {
			t.Error("IsAlive: the API says it is not alive")
		}
	})

	gotAccessToken := false
	t.Run("tokens", func(t *testing.T) {
		refresh, err := c.RefreshTokenClaims()
		if liveOK(t, "RefreshTokenClaims", err) {
			if !refresh.IsRefreshToken() {
				t.Errorf("RefreshTokenClaims: tokenType does not name a refresh token")
			}
			if api, err := refresh.APIType(); err != nil || api != want {
				t.Errorf("RefreshTokenClaims: the token is not for the API under test")
			}
			if refresh.IsExpired() {
				t.Errorf("RefreshTokenClaims: the refresh token has expired")
			}
			days := int(refresh.ExpiresIn().Hours() / 24)
			t.Logf("refresh token: expires in %d days", days)
			if !refresh.ExpiresAt.IsZero() && days < 30 {
				t.Logf("refresh token: renew it at eloverblik.dk within %d days", days)
			}
		}

		access, err := c.DataAccessTokenClaims()
		if !liveOK(t, "DataAccessTokenClaims", err) {
			return
		}
		gotAccessToken = true
		if !access.IsDataAccessToken() {
			t.Errorf("DataAccessTokenClaims: tokenType does not name a data access token")
		}
		if left := access.ExpiresIn(); left <= 0 || left > 25*time.Hour {
			t.Errorf("DataAccessTokenClaims: expires in %s, not within the 24 hours a data access token lasts", left.Round(time.Minute))
		}
	})

	return gotAccessToken
}

// liveMeteringPointReads runs the read checks both APIs share on the given metering points.
// It reads time series only for the metering points the details report as connected,
// since one that is new or disconnected has no readings, and returns those.
func liveMeteringPointReads(t *testing.T, c *client, rec *liveRecorder, ids []string, from, to time.Time) []string {
	connected := ids
	t.Run("details", func(t *testing.T) {
		if fromDetails, ok := liveDetails(t, c, rec, ids); ok {
			connected = fromDetails
		}
	})
	if skipped := len(ids) - len(connected); skipped > 0 {
		t.Logf("reading time series of %d metering points, skipping %d that are not connected", len(connected), skipped)
	}

	t.Run("time series", func(t *testing.T) {
		if len(connected) == 0 {
			t.Skip("no connected metering point")
		}
		liveTimeSeries(t, c, rec, connected, from, to)
	})
	t.Run("charges", func(t *testing.T) { liveCharges(t, c, rec, ids) })
	t.Run("charge links", func(t *testing.T) { liveChargeLinks(t, c, rec, ids, from, to) })

	return connected
}

// liveDetails checks the master data of the metering points, and returns those that are
// connected. ok is false when the call failed, and there is no answer to go by.
func liveDetails(t *testing.T, c *client, rec *liveRecorder, ids []string) (connected []string, ok bool) {
	what := "GetMeteringPointDetails"
	details, err := c.GetMeteringPointDetails(ids)
	if !liveOK(t, what, err) {
		return nil, false
	}
	liveFields[struct {
		Result []MeteringPointDetailsResponse `json:"result"`
	}](t, rec, what)

	var answered []string
	for i, item := range details {
		answered = append(answered, item.ID)

		dw := fmt.Sprintf("%s: result %d", what, i)
		if !liveItemOK(t, dw, item.StatusResponse) {
			continue
		}
		d := item.Result

		liveGSRN(t, dw, "meteringPointId", d.MeteringPointID)
		if d.MeteringPointID != item.ID {
			t.Errorf("%s: meteringPointId is not the ID the result is for", dw)
		}
		liveFilled(t, dw, map[string]string{
			"typeOfMP":                       d.TypeOfMP,
			"energyTimeSeriesMeasureUnit":    d.EnergyTimeSeriesMeasureUnit,
			"gridOperatorName":               d.GridOperatorName,
			"gridOperatorID":                 d.GridOperatorID,
			"meteringGridAreaIdentification": d.MeteringGridAreaIdentification,
			"physicalStatusOfMP":             d.PhysicalStatusOfMP,
		})
		liveMatches(t, dw, "typeOfMP", d.TypeOfMP, liveTypeOfMP)
		liveMatches(t, dw, "meteringGridAreaIdentification", d.MeteringGridAreaIdentification, liveThreeDigits)
		liveMatches(t, dw, "consumerCVR", d.ConsumerCVR, liveCVR)
		liveMatches(t, dw, "dataAccessCVR", d.DataAccessCVR, liveCVR)
		liveMatches(t, dw, "darReference", d.DarReference, liveUUID)
		liveParty(t, dw, "gridOperatorID", d.GridOperatorID, d.GridOperatorIDSchemeAgencyID)
		liveParty(t, dw, "balanceSupplierId", d.BalanceSupplierID, d.BalanceSupplierIDSchemeAgencyID)
		liveAddress{d.StreetName, d.BuildingNumber, d.Postcode, d.CityName, d.MunicipalityCode, d.LocationDescription}.check(t, dw)
		liveChildren(t, dw, d.MeteringPointID, d.ChildMeteringPoints)

		if d.PhysicalStatusOfMP == liveConnected && slices.Contains(ids, item.ID) && !slices.Contains(connected, item.ID) {
			connected = append(connected, item.ID)
		}

		// MeteringPointDetail documents these as retired, unavailable or not used. A field
		// that is set again is not an error, but the documentation needs a look.
		documentedEmpty := map[string]bool{
			"settlementMethod":           d.SettlementMethod != "",
			"consumerCategory":           d.ConsumerCategory != "",
			"estimatedAnnualVolume":      d.EstimatedAnnualVolume != "",
			"meterCounterDigits":         d.MeterCounterDigits != "",
			"meterCounterMultiplyFactor": d.MeterCounterMultiplyFactor != "",
			"meterCounterUnit":           d.MeterCounterUnit != "",
			"consumerStartDate":          !d.ConsumerStartDate.IsZero(),
			"balanceSupplierStartDate":   !d.BalanceSupplierStartDate.IsZero(),
			"taxSettlementDate":          !d.TaxSettlementDate.IsZero(),
			"mpRelationType":             d.MpRelationType != "",
		}
		if c.apiType == ThirdPartyApi {
			documentedEmpty["balanceSupplierName"] = d.BalanceSupplierName != ""
			documentedEmpty["balanceSupplierId"] = d.BalanceSupplierID != ""
		}
		var set []string
		for name, isSet := range documentedEmpty {
			if isSet {
				set = append(set, name)
			}
		}
		if len(set) > 0 {
			slices.Sort(set)
			t.Logf("%s: set, though documented as empty: %s", dw, strings.Join(set, ", "))
		}
	}
	liveSameIDs(t, what, ids, answered)

	return connected, true
}

// liveFixedStep is the length of a point in a period of a sub-day resolution, zero for
// the resolutions whose points are not of a fixed length.
func liveFixedStep(resolution Resolution) time.Duration {
	switch resolution {
	case PT15M:
		return 15 * time.Minute
	case PT1H:
		return time.Hour
	default:
		return 0
	}
}

var liveResolutions = []Resolution{PT15M, PT1H, PT1D, P1D, P1M, PT1Y, P1Y, PXD}

func liveTimeSeries(t *testing.T, c *client, rec *liveRecorder, ids []string, from, to time.Time) {
	for _, aggregation := range []Aggregation{Actual, Hour, Day} {
		what := fmt.Sprintf("GetTimeSeries(%s)", aggregation)

		tss, err := c.GetTimeSeries(ids, from, to, aggregation)
		if !liveOK(t, what, err) {
			continue
		}
		liveFields[struct {
			Result []TimeSeries `json:"result"`
		}](t, rec, what)

		var answered []string
		resolutions := map[Resolution]int{}
		for i, ts := range tss {
			answered = append(answered, ts.ID)

			tw := fmt.Sprintf("%s: result %d", what, i)
			if !liveItemOK(t, tw, ts.StatusResponse) {
				continue
			}
			liveTimeSeriesDocument(t, tw, ts.MyEnergyDataMarketDocument, from, to, resolutions)

			var sum float64
			for _, point := range ts.Flatten() {
				sum += point.Measurement
			}
			if sum == 0 {
				t.Errorf("%s: every reading of the week is 0", tw)
			}
		}
		liveSameIDs(t, what, ids, answered)

		t.Logf("%s: %d results, periods per resolution %v", what, len(tss), resolutions)
	}
}

func liveTimeSeriesDocument(t *testing.T, what string, doc MyEnergyDataMarketDocumentResponse, from, to time.Time, resolutions map[Resolution]int) {
	t.Helper()

	liveFilled(t, what, map[string]string{"mRID": doc.MRID})
	if doc.CreatedDateTime.IsZero() {
		t.Errorf("%s: createdDateTime is empty", what)
	}
	if start, end := doc.PeriodTimeInterval.Start, doc.PeriodTimeInterval.End; !start.Equal(from) || !end.Equal(to) {
		t.Errorf("%s: period.timeInterval is not the requested [from, to): its start is off by %s, its end by %s",
			what, start.Sub(from), end.Sub(to))
	}
	if len(doc.TimeSeries) == 0 {
		t.Errorf("%s: no TimeSeries", what)
	}

	for j, series := range doc.TimeSeries {
		sw := fmt.Sprintf("%s: series %d", what, j)
		liveGSRN(t, sw, "MarketEvaluationPoint.mRID.name", series.MarketEvaluationPoint.MRID.Name)
		liveFilled(t, sw, map[string]string{
			"measurement_Unit.name": series.MeasurementUnitName,
			"businessType":          series.BusinessType,
			"curveType":             series.CurveType,
		})
		if len(series.Periods) == 0 {
			t.Errorf("%s: no periods", sw)
		}

		for k, period := range series.Periods {
			pw := fmt.Sprintf("%s: period %d", sw, k)
			resolution := Resolution(period.Resolution)
			resolutions[resolution]++

			if !slices.Contains(liveResolutions, resolution) {
				t.Errorf("%s: unknown resolution %q", pw, period.Resolution)
			}
			start, end := period.TimeInterval.Start, period.TimeInterval.End
			if !start.Before(end) {
				t.Errorf("%s: timeInterval does not end after it starts", pw)
				continue
			}
			if start.Before(from) || end.After(to) {
				t.Errorf("%s: timeInterval reaches outside the requested range", pw)
			}
			if len(period.Points) == 0 {
				t.Errorf("%s: no points", pw)
				continue
			}

			livePoints(t, pw, period, liveFixedStep(resolution))
		}
	}
}

// livePoints checks the points of one period. Each must have a quality, and in a sub-day
// resolution a position inside the period that no other point has, and Flatten must give
// it the interval its position stands for. Negative quantities and positions without a
// point are logged.
func livePoints(t *testing.T, what string, period PeriodResponse, step time.Duration) {
	t.Helper()

	start, end := period.TimeInterval.Start, period.TimeInterval.End
	positions := 0
	if step > 0 {
		positions = int(end.Sub(start) / step)
	}

	seen := map[int]bool{}
	var noQuality, negative, outside, twice, misplaced int
	for _, point := range period.Points {
		if point.OutQuantityQuality == "" {
			noQuality++
		}
		if point.OutQuantityQuantity < 0 {
			negative++
		}
		if step == 0 {
			continue
		}
		if point.Position < 1 || point.Position > positions {
			outside++
			continue
		}
		if seen[point.Position] {
			twice++
		}
		seen[point.Position] = true

		from, to := pointInterval(Resolution(period.Resolution), period.TimeInterval, point.Position, len(period.Points))
		wantFrom := start.Add(time.Duration(point.Position-1) * step)
		if !from.Equal(wantFrom) || !to.Equal(wantFrom.Add(step)) {
			misplaced++
		}
	}

	for name, count := range map[string]int{
		"points without a quality":                        noQuality,
		"points with a position outside the period":       outside,
		"positions that come twice":                       twice,
		"points Flatten places off their position's slot": misplaced,
	} {
		if count > 0 {
			t.Errorf("%s: %d %s", what, count, name)
		}
	}
	if negative > 0 {
		t.Logf("%s: %d negative quantities", what, negative)
	}
	if step > 0 && len(seen) < positions {
		t.Logf("%s: %d of the period's %d points", what, len(seen), positions)
	}
}

func liveCharges(t *testing.T, c *client, rec *liveRecorder, ids []string) {
	var answered []string

	switch c.apiType {
	case CustomerApi:
		what := "GetCustomerCharges"
		charges, err := c.GetCustomerCharges(ids)
		if !liveOK(t, what, err) {
			return
		}
		liveFields[struct {
			Result []CustomerChargeResponse `json:"result"`
		}](t, rec, what)
		for i, item := range charges {
			answered = append(answered, item.ID)
			cw := fmt.Sprintf("%s: result %d", what, i)
			if liveItemOK(t, cw, item.StatusResponse) {
				r := item.Result
				liveChargeSet(t, cw, r.MeteringPointID, r.Subscriptions, r.Fees, r.Tariffs)
			}
		}
		liveSameIDs(t, what, ids, answered)

	case ThirdPartyApi:
		what := "GetThirdPartyCharges"
		charges, err := c.GetThirdPartyCharges(ids)
		if !liveOK(t, what, err) {
			return
		}
		liveFields[struct {
			Result []ThirdPartyChargeResponse `json:"result"`
		}](t, rec, what)
		for i, item := range charges {
			answered = append(answered, item.ID)
			cw := fmt.Sprintf("%s: result %d", what, i)
			if liveItemOK(t, cw, item.StatusResponse) {
				r := item.Result
				liveChargeSet(t, cw, r.MeteringPointID, r.Subscriptions, nil, r.Tariffs)
			}
		}
		liveSameIDs(t, what, ids, answered)
	}
}

// liveChargeSet checks the charges of one metering point: it has a subscription or a
// tariff, every charge has a name, an owner GLN or EIC, a price ID, a period type and a
// start date, and every tariff has prices. Negative prices are logged.
func liveChargeSet(t *testing.T, what, meteringPointID string, subscriptions, fees []Charge, tariffs []TariffCharge) {
	t.Helper()

	liveGSRN(t, what, "meteringPointId", meteringPointID)
	if len(subscriptions) == 0 && len(tariffs) == 0 {
		t.Errorf("%s: neither subscriptions nor tariffs", what)
	}

	negative := 0
	for kind, list := range map[string][]Charge{"subscription": subscriptions, "fee": fees} {
		for i, charge := range list {
			cw := fmt.Sprintf("%s: %s %d", what, kind, i)
			liveFilled(t, cw, map[string]string{
				"name": charge.Name, "owner": charge.Owner, "priceId": charge.PriceID, "periodType": charge.PeriodType,
			})
			liveParty(t, cw, "owner", charge.Owner, "")
			if charge.ValidFromDate.IsZero() {
				t.Errorf("%s: validFromDate is empty", cw)
			}
			if charge.Price < 0 {
				negative++
			}
		}
	}

	for i, tariff := range tariffs {
		tw := fmt.Sprintf("%s: tariff %d", what, i)
		liveFilled(t, tw, map[string]string{
			"name": tariff.Name, "owner": tariff.Owner, "priceId": tariff.PriceID, "periodType": tariff.PeriodType,
		})
		liveParty(t, tw, "owner", tariff.Owner, "")
		if tariff.ValidFromDate.IsZero() {
			t.Errorf("%s: validFromDate is empty", tw)
		}
		if len(tariff.Prices) == 0 {
			t.Errorf("%s: no prices", tw)
		}
		for j, price := range tariff.Prices {
			if price.Position == "" {
				t.Errorf("%s: price %d has no position", tw, j)
			}
			if price.Price < 0 {
				negative++
			}
		}
	}

	if negative > 0 {
		t.Logf("%s: %d negative prices", what, negative)
	}
}

// liveChargeLinks expects the 404 Energinet answers while the Charges integration feature
// is disabled. If the endpoint answers, README.md, llms.md, the godoc and the CLI help all
// say otherwise, and the result's shape is checked.
func liveChargeLinks(t *testing.T, c *client, rec *liveRecorder, ids []string, from, to time.Time) {
	what := "GetChargeLinksWithCharges"

	links, err := c.GetChargeLinksWithCharges(ids, from, to)
	if apiErr, ok := errors.AsType[*APIError](err); ok && apiErr.StatusCode == http.StatusNotFound {
		t.Logf("%s: 404, as documented while the Charges integration feature is disabled", what)
		return
	}
	if !liveOK(t, what, err) {
		return
	}

	t.Errorf("%s: the endpoint answers now; README.md, llms.md, the godoc and the CLI help still say it answers 404", what)
	liveFields[struct {
		Result ChargeLinksWithChargesResponse `json:"result"`
	}](t, rec, what)
	for i, result := range links.Results {
		rw := fmt.Sprintf("%s: result %d", what, i)
		liveGSRN(t, rw, "meteringPointId", result.MeteringPointID)
		if result.Error != "" {
			t.Errorf("%s: error is set", rw)
		}
	}
}

// liveNotNames returns the positions of the fields that do not look like a column name:
// letters, digits and punctuation, without a run of four digits, as an ID, a date or a
// postcode has.
func liveNotNames(record []string) []int {
	var positions []int
	for i, field := range record {
		if !liveColumnName.MatchString(field) || liveFourDigits.MatchString(field) {
			positions = append(positions, i)
		}
	}
	return positions
}

// liveCSV reads an export to its end and closes it. It returns the first record, without
// the byte order mark, as the header, and the rest as the rows. named reports whether every
// field of the header looks like a column name; only then are the names logged, so a first
// row of data is never logged.
func liveCSV(t *testing.T, what string, stream io.ReadCloser) (header []string, rows [][]string, named bool) {
	t.Helper()
	defer func() { _ = stream.Close() }()

	body := bufio.NewReader(stream)
	if bom, err := body.Peek(3); err != nil || !bytes.Equal(bom, []byte("\xEF\xBB\xBF")) {
		t.Logf("%s: the CSV does not open with a byte order mark", what)
	} else {
		_, _ = body.Discard(3)
	}

	r := csv.NewReader(body)
	r.Comma = ';'
	records, err := r.ReadAll()
	if err != nil {
		t.Errorf("%s: the export is not semicolon separated CSV with the same number of fields in every row: %s", what, describeShapeError(err))
		return nil, nil, false
	}
	if len(records) == 0 {
		t.Errorf("%s: the export is empty", what)
		return nil, nil, false
	}

	header, rows = records[0], records[1:]
	if notNames := liveNotNames(header); len(notNames) == 0 {
		named = true
		t.Logf("%s: %d columns %q, %d rows", what, len(header), header, len(rows))
	} else {
		t.Errorf("%s: the first row is not a header of column names: fields %v of %d do not look like names (%d rows)", what, notNames, len(header), len(rows))
	}
	if len(rows) == 0 {
		t.Errorf("%s: the export has a header and no rows", what)
	}
	if column := slices.Index(header, "MålepunktsID"); named && column >= 0 {
		liveExportIDs(t, what, rows, column)
	}
	return header, rows, named
}

// liveExportIDs checks the MålepunktsID column of an export: every value, without the
// white space around it, is a metering point ID. The white space is logged by where it
// sits and what it is, and a value that is no ID by its shape, never the value.
func liveExportIDs(t *testing.T, what string, rows [][]string, column int) {
	t.Helper()

	padding, badShapes := map[string]int{}, map[string]int{}
	bad := 0
	for _, row := range rows {
		value := row[column]
		trimmed := strings.TrimSpace(value)
		if trimmed != value {
			lead := value[:len(value)-len(strings.TrimLeftFunc(value, unicode.IsSpace))]
			trail := value[len(strings.TrimRightFunc(value, unicode.IsSpace)):]
			padding[fmt.Sprintf("%q before and %q after the ID", lead, trail)]++
		}
		if !validGSRN(trimmed) {
			bad++
			badShapes[liveShape(trimmed)]++
		}
	}

	for where, count := range padding {
		t.Logf("%s: %d MålepunktsID values with %s", what, count, where)
	}
	if bad > 0 {
		t.Errorf("%s: %d rows whose MålepunktsID is not a metering point ID", what, bad)
		for shape, count := range badShapes {
			t.Logf("%s: %d MålepunktsID values of %s", what, count, shape)
		}
	}
}

// liveExport reads an export and checks its columns against the documented ones.
func liveExport(t *testing.T, what string, columns []string, export func() (io.ReadCloser, error)) {
	t.Helper()
	stream, err := export()
	if !liveOK(t, what, err) {
		return
	}
	if header, _, named := liveCSV(t, what, stream); header != nil {
		liveColumns(t, what, header, named, columns)
	}
}

// liveColumns reports a header that is not the documented columns, naming the documented
// columns it lacks and, when the header holds column names, the columns it adds.
func liveColumns(t *testing.T, what string, header []string, named bool, columns []string) bool {
	t.Helper()
	if slices.Equal(header, columns) {
		return true
	}

	var missing, unexpected []string
	for _, name := range columns {
		if !slices.Contains(header, name) {
			missing = append(missing, name)
		}
	}
	if named {
		for _, name := range header {
			if !slices.Contains(columns, name) {
				unexpected = append(unexpected, name)
			}
		}
	}
	t.Errorf("%s: %d columns where llms.md documents %d; missing %q, not documented %q", what, len(header), len(columns), missing, unexpected)
	return false
}

// liveExportTimeSeries checks the documented columns of a time series export, and that
// every row has two timestamps and a quantity, not all of them 0. It exports the days of
// [from, to) as the docs say to, with to minus one day, since the export includes to: the
// rows must reach into that last day and stop at its end, so an export that dropped or
// overran its to fails either way. A row without a quantity, as for an hour the grid
// operator reported missing, is logged.
func liveExportTimeSeries(t *testing.T, c *client, ids []string, from, to time.Time) {
	t.Helper()

	what := "ExportTimeSeries(Hour)"
	if len(ids) == 0 {
		t.Logf("%s: skipped, no connected metering point", what)
		return
	}
	lastDay := to.AddDate(0, 0, -1)
	stream, err := c.ExportTimeSeries(ids, from, lastDay, Hour)
	if !liveOK(t, what, err) {
		return
	}
	header, rows, named := liveCSV(t, what, stream)
	if header == nil || !liveColumns(t, what, header, named, liveExportColumns) {
		return
	}

	var badTime, badQuantity, noQuantity int
	var first, last time.Time
	var sum float64
	for _, row := range rows {
		rowFrom, errFrom := time.ParseInLocation("02-01-2006 15:04:05", row[1], cph)
		rowTo, errTo := time.ParseInLocation("02-01-2006 15:04:05", row[2], cph)
		if errFrom != nil || errTo != nil {
			badTime++
		} else {
			if first.IsZero() || rowFrom.Before(first) {
				first = rowFrom
			}
			if rowTo.After(last) {
				last = rowTo
			}
		}
		if row[3] == "" {
			noQuantity++
			continue
		}
		quantity, err := strconv.ParseFloat(strings.ReplaceAll(row[3], ",", "."), 64)
		if err != nil {
			badQuantity++
		}
		sum += quantity
	}

	for name, count := range map[string]int{
		"Fra_dato or Til_dato values not in DD-MM-YYYY hh:mm:ss": badTime,
		"rows whose Mængde is not a decimal number":              badQuantity,
	} {
		if count > 0 {
			t.Errorf("%s: %d %s", what, count, name)
		}
	}
	if noQuantity > 0 {
		t.Logf("%s: %d rows without a Mængde", what, noQuantity)
	}
	// Asked for from through lastDay, the export's rows run from from to the end of lastDay,
	// which is to
	if !first.IsZero() && first.Before(from) {
		t.Errorf("%s: the rows start %s before the requested from", what, from.Sub(first))
	}
	if !last.IsZero() && last.After(to) {
		t.Errorf("%s: the rows end %s after the end of the requested to", what, last.Sub(to))
	}
	if !last.IsZero() && !last.After(lastDay) {
		t.Errorf("%s: no row reaches into the requested to; the export no longer includes to", what)
	}
	if !first.IsZero() {
		t.Logf("%s: the rows cover %s of the %s from from through to", what, last.Sub(first), to.Sub(from))
	}
	if len(rows) > 0 && sum == 0 {
		t.Errorf("%s: every Mængde of the week is 0", what)
	}
}
