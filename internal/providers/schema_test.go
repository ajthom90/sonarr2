package providers

import (
	"testing"
)

// TestSchemaForGeneratesFields verifies that a struct with three tagged fields
// produces exactly three FieldSchema entries with the correct types and labels.
func TestSchemaForGeneratesFields(t *testing.T) {
	type Settings struct {
		URL    string `form:"text"     label:"URL"     required:"true"`
		APIKey string `form:"password" label:"API Key"`
		Port   int    `form:"number"   label:"Port"    default:"8080"`
	}

	s := SchemaFor(Settings{})

	if len(s.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(s.Fields))
	}

	cases := []struct {
		name      string
		wantType  string
		wantLabel string
	}{
		{"URL", "text", "URL"},
		{"APIKey", "password", "API Key"},
		{"Port", "number", "Port"},
	}
	for i, c := range cases {
		f := s.Fields[i]
		if f.Name != c.name {
			t.Errorf("field[%d].Name = %q, want %q", i, f.Name, c.name)
		}
		if f.Type != c.wantType {
			t.Errorf("field[%d].Type = %q, want %q", i, f.Type, c.wantType)
		}
		if f.Label != c.wantLabel {
			t.Errorf("field[%d].Label = %q, want %q", i, f.Label, c.wantLabel)
		}
	}
	if !s.Fields[0].Required {
		t.Error("URL field should be required")
	}
	if s.Fields[1].Required {
		t.Error("APIKey field should not be required")
	}
	if s.Fields[2].Default != "8080" {
		t.Errorf("Port.Default = %q, want %q", s.Fields[2].Default, "8080")
	}
}

// TestSchemaForSkipsUntaggedFields verifies that fields without a "form" tag
// are omitted from the resulting schema.
func TestSchemaForSkipsUntaggedFields(t *testing.T) {
	type Settings struct {
		Visible string `form:"text" label:"Visible"`
		Hidden  string // no form tag
		Also    int    // no form tag
	}

	s := SchemaFor(Settings{})

	if len(s.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(s.Fields))
	}
	if s.Fields[0].Name != "Visible" {
		t.Errorf("unexpected field name %q", s.Fields[0].Name)
	}
}

// TestSchemaForHandlesAllTypes verifies that text, password, number, checkbox,
// and select form types are all preserved correctly, along with optional
// metadata such as placeholder, helpText, and advanced.
func TestSchemaForHandlesAllTypes(t *testing.T) {
	type Settings struct {
		Host      string `form:"text"        label:"Host"       placeholder:"localhost"`
		Pass      string `form:"password"    label:"Password"`
		Timeout   int    `form:"number"      label:"Timeout"    default:"30"`
		UseTLS    bool   `form:"checkbox"    label:"Use TLS"`
		OutputDir string `form:"select"      label:"Output Dir" helpText:"Where to save files" advanced:"true"`
	}

	s := SchemaFor(Settings{})

	if len(s.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(s.Fields))
	}

	wantTypes := []string{"text", "password", "number", "checkbox", "select"}
	for i, wt := range wantTypes {
		if s.Fields[i].Type != wt {
			t.Errorf("field[%d].Type = %q, want %q", i, s.Fields[i].Type, wt)
		}
	}

	if s.Fields[0].Placeholder != "localhost" {
		t.Errorf("Host.Placeholder = %q, want %q", s.Fields[0].Placeholder, "localhost")
	}
	if s.Fields[2].Default != "30" {
		t.Errorf("Timeout.Default = %q, want %q", s.Fields[2].Default, "30")
	}
	if s.Fields[4].HelpText != "Where to save files" {
		t.Errorf("OutputDir.HelpText = %q", s.Fields[4].HelpText)
	}
	if !s.Fields[4].Advanced {
		t.Error("OutputDir should be advanced")
	}
}
