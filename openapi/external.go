package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/dolmen-go/jsonptr"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

var interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
var stringType = reflect.TypeOf("")
var stringInterfaceMapType = reflect.MapOf(stringType, interfaceType)

// YAML serializers are really, really, really annoying in that
// it decodes maps into map[interface{}]interface{} instead
// of map[string]interfaace{}
func restoreSanity(rv reflect.Value) reflect.Value {
	rv, _ = restoreSanityInternal(rv)
	return rv
}

func stringify(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}

	return `(invalid)`
}

func restoreSanityInternal(rv reflect.Value) (reflect.Value, bool) {
	if rv.Kind() == reflect.Interface {
		return restoreSanityInternal(rv.Elem())
	}

	switch rv.Kind() {
	case reflect.Map:
		// the keys MUST Be strings.
		if rv.Type().Key().Kind() == reflect.String {
			return rv, false
		}
		newMap := reflect.MakeMap(stringInterfaceMapType)
		for _, key := range rv.MapKeys() {
			newValue, _ := restoreSanityInternal(rv.MapIndex(key))
			newMap.SetMapIndex(reflect.ValueOf(stringify(key.Elem().Interface())), newValue)
		}
		return newMap, true
	case reflect.Slice, reflect.Array:
		var count int
		for i := 0; i < rv.Len(); i++ {
			newValue, restored := restoreSanityInternal(rv.Index(i))
			if restored {
				rv.Index(i).Set(newValue)
				count++
			}
		}
		return rv, count > 0
	default:
		return rv, false
	}
}

var zeroval reflect.Value
var refKey = reflect.ValueOf(`$ref`)

func parseRef(s string) (string, string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", "", errors.Wrapf(err, `failed to parse URL %s`, s)
	}

	frag := u.Fragment
	u.Fragment = ""
	return u.String(), frag, nil
}

func isExternal(s string) bool {
	if strings.HasPrefix(s, `google/protobuf/`) {
		return false
	}
	return strings.IndexByte(s, '#') != 0
}

func NewResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) Resolve(v interface{}, options ...Option) (interface{}, error) {
	var dir string
	for _, o := range options {
		switch o.Name() {
		case optkeyDir:
			dir = o.Value().(string)
		}
	}

	c := resolveCtx{
		dir:                dir,
		externalReferences: map[string]interface{}{},
		cache:              map[string]interface{}{},
	}

	rv, err := c.resolve(restoreSanity(reflect.ValueOf(v)))
	if err != nil {
		return nil, errors.Wrap(err, `failed to resolve object`)
	}

	return restoreSanity(rv).Interface(), nil
}

// note, we must use a composite type with only map[string]interface{},
// []interface{} and interface{} as its building blocks
func (c *resolveCtx) resolve(rv reflect.Value) (reflect.Value, error) {
	if rv.Kind() == reflect.Interface {
		return c.resolve(rv.Elem())
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			newV, err := c.resolve(rv.Index(i))
			if err != nil {
				return zeroval, errors.Wrapf(err, `failed to resolve element %d`, i)
			}
			rv.Index(i).Set(newV)
		}
	case reflect.Map:
		// if it's a map, see if we have a "$ref" key
		if refValue := rv.MapIndex(refKey); refValue != zeroval {
			if refValue.Kind() != reflect.Interface {
				return zeroval, errors.Errorf("'$ref' key contains non-interface{} element (%s)", refValue.Type())
			}
			refValue = refValue.Elem()

			if refValue.Kind() != reflect.String {
				return zeroval, errors.Errorf("'$ref' key contains non-string element (%s)", refValue.Type())
			}

			ref := refValue.String()
			if isExternal(ref) {
				refURL, refFragment, err := parseRef(ref)
				if err != nil {
					return zeroval, errors.Wrap(err, `failed to parse reference`)
				}

				// if we have already loaded this, don't make another
				// roundtrip to the remote server
				resolved, ok := c.cache[refURL]
				if !ok {
					var err error
					resolved, err = c.loadExternal(refURL)
					if err != nil {
						return zeroval, errors.Wrapf(err, `failed to resolve external reference %s`, ref)
					}
					// remember that we have resolved this document
					c.cache[refURL] = resolved
				}

				docFragment, err := jsonptr.Get(restoreSanity(reflect.ValueOf(resolved)).Interface(), refFragment)
				if err != nil {
					return zeroval, errors.Wrapf(err, `failed to resolve document fragment %s`, refFragment)
				}

				// recurse into docFragment
				return c.resolve(reflect.ValueOf(docFragment))
			}
			return rv, nil
		}

		// otherwise, traverse the map
		for _, key := range rv.MapKeys() {
			newV, err := c.resolve(rv.MapIndex(key))
			if err != nil {
				return zeroval, errors.Wrapf(err, `failed to resolve map element for %s`, key)
			}
			rv.SetMapIndex(key, newV)
		}
		return rv, nil
	}
	return rv, nil
}

func (c *resolveCtx) normalizePath(s string) string {
	if c.dir == "" {
		return s
	}
	return filepath.Join(c.dir, s)
}

func (c *resolveCtx) loadExternal(s string) (interface{}, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to parse reference %s`, s)
	}

	var src io.Reader
	switch u.Scheme {
	case "":
		fmt.Fprintf(os.Stdout, "loading local file %s\n", u.Path)
		f, err := os.Open(c.normalizePath(u.Path))
		if err != nil {
			return nil, errors.Wrapf(err, `failed to read local file %s`, u.Path)
		}
		defer f.Close()
		src = f
	case "http", "https":
		fmt.Fprintf(os.Stdout, "Fetching %s\n", u.String())
		res, err := http.Get(u.String())
		if err != nil {
			return nil, errors.Wrapf(err, `failed to fetch remote file %s`, u.String())
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, errors.Wrapf(err, `failed to fetch remote file %s`, u.String())
		}

		src = res.Body
	default:
		return nil, errors.Errorf(`cannot handle reference %s`, s)
	}

	// now guess from the file nam if this is a YAML or JSON
	var v interface{}
	switch strings.ToLower(path.Ext(u.Path)) {
	case ".yaml", ".yml":
		if err := yaml.NewDecoder(src).Decode(&v); err != nil {
			return nil, errors.Wrapf(err, `failed to decode reference %s`, s)
		}
	default:
		if err := json.NewDecoder(src).Decode(&v); err != nil {
			return nil, errors.Wrapf(err, `failed to decode reference %s`, s)
		}
	}

	return v, nil
}
