package eloverblik

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The helpers in this file check the shape of an API response rather than its values. The
// live tests (live_test.go, behind the live build tag) use them on the production API; the
// tests below keep them honest in every run.

// validGS1 reports whether s is n digits, the last of them the GS1 check digit of the
// others, as in an 18-digit GSRN or a 13-digit GLN.
func validGS1(s string, n int) bool {
	if len(s) != n {
		return false
	}

	sum := 0
	for i := range n {
		c := s[n-1-i]
		if c < '0' || c > '9' {
			return false
		}
		// Counted from the right, the check digit is position 0 and the data digits next
		// to it weigh 3, 1, 3, 1, ...
		if i > 0 {
			weight := 1
			if i%2 == 1 {
				weight = 3
			}
			sum += int(c-'0') * weight
		}
	}

	return int(s[n-1]-'0') == (10-sum%10)%10
}

// validGSRN reports whether s has the format of a Danish metering point ID: an 18-digit
// GSRN under the Danish GS1 prefix 57, with a valid check digit.
func validGSRN(s string) bool {
	return strings.HasPrefix(s, "57") && validGS1(s, 18)
}

// jsonField is a key a struct type decodes from, and the type of the field it fills.
type jsonField struct {
	name string
	typ  reflect.Type
}

// jsonFields lists the keys a struct type decodes, as encoding/json sees them: a field's
// tag name or else its name, leaving out the fields tagged "-" and the unexported ones,
// and promoting the fields of an untagged embedded struct.
func jsonFields(t reflect.Type) []jsonField {
	var fields []jsonField

	for f := range t.Fields() {
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")

		if f.Anonymous && name == "" {
			embedded := f.Type
			if embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				fields = append(fields, jsonFields(embedded)...)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		fields = append(fields, jsonField{name: name, typ: f.Type})
	}

	return fields
}

// matchField finds the field a key decodes into: the one of that exact name, or else one
// whose name matches it ignoring case, as encoding/json does.
func matchField(fields []jsonField, key string) (jsonField, bool) {
	if i := slices.IndexFunc(fields, func(f jsonField) bool { return f.name == key }); i >= 0 {
		return fields[i], true
	}
	if i := slices.IndexFunc(fields, func(f jsonField) bool { return strings.EqualFold(f.name, key) }); i >= 0 {
		return fields[i], true
	}
	return jsonField{}, false
}

var jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()

// fieldDriftWalker walks a decoded JSON value alongside the Go type it decodes into. All
// three sets hold paths such as "result[].isMovedOut".
type fieldDriftWalker struct {
	unknown  map[string]bool // keys no field matches
	declared map[string]bool // fields of every object met
	seen     map[string]bool // fields at least one object carried
}

func (w *fieldDriftWalker) walk(v any, t reflect.Type, path string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// A type that decodes itself, such as FlexibleTime or time.Time, is a leaf.
	if reflect.PointerTo(t).Implements(jsonUnmarshalerType) {
		return
	}

	switch t.Kind() {
	case reflect.Struct:
		obj, ok := v.(map[string]any)
		if !ok {
			return
		}
		fields := jsonFields(t)
		for _, f := range fields {
			w.declared[joinPath(path, f.name)] = true
		}
		for key, val := range obj {
			f, ok := matchField(fields, key)
			if !ok {
				w.unknown[joinPath(path, key)] = true
				continue
			}
			fieldPath := joinPath(path, f.name)
			w.seen[fieldPath] = true
			w.walk(val, f.typ, fieldPath)
		}

	case reflect.Slice, reflect.Array:
		items, ok := v.([]any)
		if !ok {
			return
		}
		for _, item := range items {
			w.walk(item, t.Elem(), path+"[]")
		}

	case reflect.Map:
		obj, ok := v.(map[string]any)
		if !ok {
			return
		}
		for _, val := range obj {
			w.walk(val, t.Elem(), path+"{}")
		}
	}
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// jsonFieldDrift compares the JSON objects in body with the Go type body decodes into. It
// returns the keys no field of the type matches, which decoding drops, and the fields no
// object carried, both sorted. A key with a null value counts as carried. Keys match
// fields ignoring case, as in encoding/json. A type that decodes itself is not looked into.
func jsonFieldDrift(body []byte, typ reflect.Type) (unknown, missing []string, err error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, nil, err
	}

	w := fieldDriftWalker{unknown: map[string]bool{}, declared: map[string]bool{}, seen: map[string]bool{}}
	w.walk(v, typ, "")

	for path := range w.declared {
		if !w.seen[path] {
			missing = append(missing, path)
		}
	}
	slices.Sort(missing)

	return slices.Sorted(maps.Keys(w.unknown)), missing, nil
}

var (
	shapeLongNumber = regexp.MustCompile(`\d{8,}`)
	shapeUUID       = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	shapeQuoted     = regexp.MustCompile(`"[^"]*"`)
)

// describeShapeError renders an error without the values it may quote from a response or
// a request, so the output of the live tests holds no customer data. The errors that quote
// a value are rendered from their fields instead of their message: a decode error names
// the field and the type but not the value, a time or number parse error the layout or
// function, a transport error leaves out the URL, which can hold an authorization ID, and
// a failed metering point leaves out its ID. What remains is masked as well: quoted
// strings, UUIDs and every number of eight digits or more.
func describeShapeError(err error) string {
	msg := err.Error()
	if quoted, safe, ok := safeErrorText(err); ok {
		msg = strings.Replace(msg, quoted, safe, 1)
	}
	msg = shapeQuoted.ReplaceAllString(msg, `"…"`)
	msg = shapeUUID.ReplaceAllString(msg, "<uuid>")
	return shapeLongNumber.ReplaceAllString(msg, "<number>")
}

// safeErrorText finds the first error in the chain that quotes a value, and returns its
// message and the text to put in its place.
func safeErrorText(err error) (quoted, safe string, ok bool) {
	if e, ok := errors.AsType[*url.Error](err); ok {
		return e.Error(), e.Op + " <url>: " + describeShapeError(e.Err), true
	}
	if e, ok := errors.AsType[*itemError](err); ok {
		return e.Error(), fmt.Sprintf("eloverblik: metering point <id>: %d %s", e.code, e.text), true
	}
	if e, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		return e.Error(), fmt.Sprintf("json: cannot decode %s into %s", e.Field, e.Type), true
	}
	if e, ok := errors.AsType[*json.SyntaxError](err); ok {
		return e.Error(), fmt.Sprintf("json: syntax error at offset %d", e.Offset), true
	}
	if e, ok := errors.AsType[*time.ParseError](err); ok {
		return e.Error(), "time: a value does not match the layout " + e.Layout, true
	}
	if e, ok := errors.AsType[*strconv.NumError](err); ok {
		return e.Error(), fmt.Sprintf("strconv.%s: %v", e.Func, e.Err), true
	}
	if e, ok := errors.AsType[*csv.ParseError](err); ok {
		return e.Error(), fmt.Sprintf("csv: line %d, column %d: %v", e.Line, e.Column, e.Err), true
	}
	return "", "", false
}

func TestValidGS1(t *testing.T) {
	assert.True(t, validGS1("571313180400000001", 18))
	assert.True(t, validGS1("5790000432752", 13))
	assert.False(t, validGS1("571313180400000002", 18), "wrong check digit")
	assert.False(t, validGS1("57131318040000000", 18), "17 digits")
	assert.False(t, validGS1("5713131804000000010", 18), "19 digits")
	assert.False(t, validGS1("57131318040000000a", 18), "not a digit")

	assert.True(t, validGSRN("571313180400000001"))
	assert.False(t, validGSRN("001313180400000003"), "valid GS1, but not under the Danish prefix")
	assert.False(t, validGSRN("5790000432752"), "a GLN")
}

func TestJSONFieldDrift(t *testing.T) {
	type inner struct {
		Code string `json:"code"`
		Gone string `json:"gone"`
	}
	type embedded struct {
		Success bool `json:"success"`
	}
	type outer struct {
		Items   []inner          `json:"items"`
		When    FlexibleTime     `json:"when"`
		ByName  map[string]inner `json:"byName"`
		Pointer *inner           `json:"pointer"`
		Ignored string           `json:"-"`
		embedded
	}

	body := `{
		"items": [{"code": "a", "extra": 1}, {"CODE": "b"}],
		"when": {"inside": "a type that decodes itself"},
		"byName": {"x": {"code": "c", "other": true}},
		"pointer": null,
		"success": true,
		"Ignored": "x",
		"new": 1
	}`

	unknown, missing, err := jsonFieldDrift([]byte(body), reflect.TypeFor[outer]())

	require.NoError(t, err)
	assert.Equal(t, []string{"Ignored", "byName{}.other", "items[].extra", "new"}, unknown)
	assert.Equal(t, []string{"byName{}.gone", "items[].gone"}, missing)

	t.Run("not JSON", func(t *testing.T) {
		_, _, err := jsonFieldDrift([]byte("<html>"), reflect.TypeFor[outer]())
		require.Error(t, err)
	})
}

func TestDescribeShapeError(t *testing.T) {
	decode := func(body string, into any) error { return json.Unmarshal([]byte(body), into) }

	cases := map[string]struct {
		err      error
		hidden   []string
		readable []string
	}{
		"a quantity the decoder rejects": {
			err: decode(`{"result":[{"MyEnergyData_MarketDocument":{"TimeSeries":[{"Period":[{"point":[{"out_Quantity.quantity":"0,734"}]}]}]}}]}`,
				&struct {
					Result []TimeSeries `json:"result"`
				}{}),
			hidden:   []string{"0,734"},
			readable: []string{"out_Quantity.quantity", "float64"},
		},
		"a date without an offset": {
			err: decode(`{"result":[{"consumerStartDate":"2019-04-01T00:00:00"}]}`, &struct {
				Result []MeteringPoints `json:"result"`
			}{}),
			hidden:   []string{"2019-04-01"},
			readable: []string{"layout"},
		},
		"a transport error on an authorization's metering points": {
			err: fmt.Errorf("request failed: %w", &url.Error{
				Op:  "Get",
				URL: "https://api.eloverblik.dk/thirdpartyapi/api/authorization/authorization/meteringpoints/authorizationId/3f2a1b4c-0000-4000-8000-000000000001",
				Err: errors.New("connection reset by peer"),
			}),
			hidden:   []string{"3f2a1b4c", "authorizationId"},
			readable: []string{"request failed", "Get <url>", "connection reset by peer"},
		},
		"a metering point that failed on its own": {
			err:      StatusResponse{ErrorCode: 30015, ErrorText: "NoDataAvailable", ID: "571313180400000001"}.Err(),
			hidden:   []string{"571313180400000001"},
			readable: []string{"30015", "NoDataAvailable"},
		},
		"an API message that quotes a metering point": {
			err:      fmt.Errorf("unhandled error: '[20008] Metering point 571313180400000001 not found'"),
			hidden:   []string{"571313180400000001"},
			readable: []string{"20008"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, tc.err)
			got := describeShapeError(tc.err)
			for _, value := range tc.hidden {
				assert.NotContains(t, got, value)
			}
			for _, part := range tc.readable {
				assert.Contains(t, got, part)
			}
		})
	}

	t.Run("a CSV row of the wrong length", func(t *testing.T) {
		r := csv.NewReader(strings.NewReader("a;b\n571313180400000001"))
		r.Comma = ';'
		_, err := r.ReadAll()
		require.Error(t, err)
		got := describeShapeError(err)
		assert.Contains(t, got, "line 2")
		assert.NotContains(t, got, "571313180400000001")
	})
}
