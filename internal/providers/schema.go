package providers

import (
	"reflect"
	"strconv"
)

// FieldSchema describes a single form field in a provider settings form.
type FieldSchema struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // text, password, number, checkbox, select, multiselect
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	HelpText    string `json:"helpText,omitempty"`
	Advanced    bool   `json:"advanced,omitempty"`
}

// Schema holds the full set of form fields for a provider settings struct.
type Schema struct {
	Fields []FieldSchema `json:"fields"`
}

// SchemaFor generates a form schema from a settings struct's field tags.
// It walks the exported fields of settings (or a pointer to a struct) and
// includes each field that carries a "form" struct tag.
//
// Supported tags: form, label, required, default, placeholder, helpText, advanced.
func SchemaFor(settings any) Schema {
	t := reflect.TypeOf(settings)
	if t == nil {
		return Schema{}
	}
	// Dereference pointer.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return Schema{}
	}

	var fields []FieldSchema
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		formType, ok := f.Tag.Lookup("form")
		if !ok || formType == "" {
			continue
		}

		fs := FieldSchema{
			Name:        f.Name,
			Type:        formType,
			Label:       f.Tag.Get("label"),
			Default:     f.Tag.Get("default"),
			Placeholder: f.Tag.Get("placeholder"),
			HelpText:    f.Tag.Get("helpText"),
		}

		if req := f.Tag.Get("required"); req != "" {
			fs.Required, _ = strconv.ParseBool(req)
		}
		if adv := f.Tag.Get("advanced"); adv != "" {
			fs.Advanced, _ = strconv.ParseBool(adv)
		}

		fields = append(fields, fs)
	}

	return Schema{Fields: fields}
}
