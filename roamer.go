// Package roamer provides flexible http request parser.
package roamer

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/slipros/exp"
	rerr "github.com/slipros/roamer/err"
	rexp "github.com/slipros/roamer/internal/experiment"
	"github.com/slipros/roamer/parser"
	"github.com/slipros/roamer/value"
)

// AfterParser will be called after http request parsing.
//
//go:generate mockery --name=AfterParser --outpkg=mock --output=./mock
type AfterParser interface {
	AfterParse(r *http.Request) error
}

// Roamer flexible http request parser.
type Roamer struct {
	parsers                     Parsers
	decoders                    Decoders
	formatters                  Formatters
	skipFilled                  bool
	hasParsers                  bool
	hasDecoders                 bool
	hasFormatters               bool
	experimentalFastStructField bool
}

// NewRoamer creates and returns new roamer.
func NewRoamer(opts ...OptionsFunc) *Roamer {
	r := Roamer{
		parsers:    make(Parsers),
		decoders:   make(Decoders),
		formatters: make(Formatters),
		skipFilled: true,
	}

	for _, opt := range opts {
		opt(&r)
	}

	r.hasParsers = len(r.parsers) > 0
	r.hasDecoders = len(r.decoders) > 0
	r.hasFormatters = len(r.formatters) > 0

	if r.experimentalFastStructField {
		r.enableExperimentalFeatures()
	}

	return &r
}

// Parse parses http request into ptr.
//
// ptr can implement AfterParser to execute some logic after parsing.
func (r *Roamer) Parse(req *http.Request, ptr any) error {
	if ptr == nil {
		return errors.Wrapf(rerr.NilValue, "ptr")
	}

	t := reflect.TypeOf(ptr)
	if t.Kind() != reflect.Pointer {
		return errors.Wrapf(rerr.NotPtr, "`%T`", ptr)
	}

	switch t.Elem().Kind() {
	case reflect.Struct:
		if err := r.parseStruct(req, ptr); err != nil {
			return err
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		if err := r.parseBody(req, ptr); err != nil {
			return err
		}
	default:
		return errors.Wrapf(rerr.NotSupported, "`%T`", ptr)
	}

	if p, ok := ptr.(AfterParser); ok {
		return p.AfterParse(req)
	}

	return nil
}

// parseStruct parses structure from http request into a ptr.
func (r *Roamer) parseStruct(req *http.Request, ptr any) error {
	if err := r.parseBody(req, ptr); err != nil {
		return err
	}

	if !r.hasParsers {
		return nil
	}

	v := reflect.Indirect(reflect.ValueOf(ptr))
	t := v.Type()

	var fieldType reflect.StructField

	fieldsAmount := v.NumField()
	cache := make(parser.Cache, fieldsAmount)

	for i := range fieldsAmount {
		if r.experimentalFastStructField {
			ft, exists := exp.FastStructField(&v, i)
			if !exists {
				// should never happen - anomaly.
				return errors.WithStack(rerr.FieldIndexOutOfBounds)
			}

			fieldType = ft
		} else {
			fieldType = t.Field(i)
		}

		if !fieldType.IsExported() || len(fieldType.Tag) == 0 {
			continue
		}

		fieldValue := v.Field(i)
		if r.skipFilled && !fieldValue.IsZero() {
			if r.hasFormatters {
				if err := r.formatFieldValue(&fieldType, fieldValue); err != nil {
					return errors.WithMessagef(err, "format field `%s` in struct `%T`", fieldType.Name, ptr)
				}
			}

			continue
		}

		for tag, p := range r.parsers {
			parsedValue, ok := p.Parse(req, fieldType.Tag, cache)
			if !ok {
				continue
			}

			if err := value.Set(fieldValue, parsedValue); err != nil {
				return errors.Wrapf(err, "set `%s` value to field `%s` from tag `%s` for struct `%T`",
					parsedValue, fieldType.Name, tag, ptr)
			}

			break
		}

		if r.hasFormatters {
			if err := r.formatFieldValue(&fieldType, fieldValue); err != nil {
				return errors.WithMessagef(err, "format field `%s` in struct `%T`", fieldType.Name, ptr)
			}
		}
	}

	return nil
}

// formatFieldValue format field value.
func (r *Roamer) formatFieldValue(fieldType *reflect.StructField, fieldValue reflect.Value) error {
	if !r.formatters.has(fieldType.Tag) {
		return nil
	}

	fieldPtrValue, ok := value.Pointer(fieldValue)
	if !ok {
		return nil
	}

	for _, f := range r.formatters {
		if err := f.Format(fieldType.Tag, fieldPtrValue); err != nil {
			return err
		}
	}

	return nil
}

// parseStruct parses body from http request into a ptr.
func (r *Roamer) parseBody(req *http.Request, ptr any) error {
	if !r.hasDecoders || req.ContentLength == 0 || req.Method == http.MethodGet {
		return nil
	}

	contentType := req.Header.Get("Content-Type")
	if base, _, found := strings.Cut(contentType, ";"); found {
		contentType = base
	}

	d, ok := r.decoders[contentType]
	if !ok {
		return nil
	}

	if err := d.Decode(req, ptr); err != nil {
		return errors.WithStack(rerr.DecodeError{
			Err: errors.WithMessagef(err, "decode `%s` request body for `%T`", contentType, ptr),
		})
	}

	return nil
}

// enableExperimentalFeatures enables experimental features.
func (r *Roamer) enableExperimentalFeatures() {
	for _, d := range r.decoders {
		e, ok := d.(rexp.Experiment)
		if !ok {
			continue
		}

		e.EnableExperimentalFastStructFieldParser()
	}
}
