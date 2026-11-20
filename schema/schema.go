package schema

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v4"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed note.schema.json
var noteSchema []byte

//go:embed strata.schema.json
var configSchema []byte

var (
	noteCompiled   = compile("note.schema.json", noteSchema)
	configCompiled = compile("strata.schema.json", configSchema)
	printer        = message.NewPrinter(language.English)
)

func compile(name string, raw []byte) *jsonschema.Schema {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		panic(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(name, doc); err != nil {
		panic(err)
	}
	return c.MustCompile(name)
}

func ValidateNote(raw []byte) []error   { return validate(noteCompiled, raw) }
func ValidateConfig(raw []byte) []error { return validate(configCompiled, raw) }

func validate(s *jsonschema.Schema, raw []byte) []error {
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return []error{err}
	}
	asJSON, err := json.Marshal(v)
	if err != nil {
		return []error{err}
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(asJSON))
	if err != nil {
		return []error{err}
	}
	err = s.Validate(inst)
	if err == nil {
		return nil
	}
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []error{err}
	}
	var errs []error
	leaves(ve, &errs)
	return errs
}

func leaves(ve *jsonschema.ValidationError, out *[]error) {
	if len(ve.Causes) == 0 {
		*out = append(*out, fmt.Errorf("/%s: %s", strings.Join(ve.InstanceLocation, "/"), ve.ErrorKind.LocalizedString(printer)))
		return
	}
	for _, c := range ve.Causes {
		leaves(c, out)
	}
}
