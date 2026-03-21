package parser

import (
	"strings"
	"testing"
)

func strVal(s string) Value  { return Value{Type: StringType, Raw: s} }
func numVal(n float64) Value { return Value{Type: NumberType, Raw: n} }
func boolVal(b bool) Value   { return Value{Type: BoolType, Raw: b} }
func listVal(items ...Value) Value {
	return Value{Type: ListType, Raw: items}
}

func TestFormatter(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{
			name: "single block no attributes",
			config: &Config{
				Blocks: []Block{
					{Type: "egg", Labels: []string{"my-app"}, Attributes: map[string]Value{}, Blocks: nil},
				},
			},
			want: `egg "my-app" {
}`,
		},
		{
			name: "attributes sorted alphabetically",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"my-app"},
						Attributes: map[string]Value{
							"zebra": strVal("last"),
							"alpha": strVal("first"),
							"mango": strVal("middle"),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "my-app" {
  alpha = "first"
  mango = "middle"
  zebra = "last"
}`,
		},
		{
			name: "nested blocks in source order",
			config: &Config{
				Blocks: []Block{
					{
						Type:       "egg",
						Labels:     []string{"my-app"},
						Attributes: map[string]Value{},
						Blocks: []Block{
							{Type: "runner", Labels: nil, Attributes: map[string]Value{"type": strVal("vm")}, Blocks: nil},
							{Type: "resources", Labels: nil, Attributes: map[string]Value{"cpu": numVal(2)}, Blocks: nil},
						},
					},
				},
			},
			want: `egg "my-app" {
  runner {
    type = "vm"
  }
  resources {
    cpu = 2
  }
}`,
		},
		{
			name: "string values double-quoted",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"x"},
						Attributes: map[string]Value{
							"name": strVal("hello world"),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "x" {
  name = "hello world"
}`,
		},
		{
			name: "list with 1 element inline",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"x"},
						Attributes: map[string]Value{
							"tags": listVal(strVal("docker")),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "x" {
  tags = ["docker"]
}`,
		},
		{
			name: "list with 2 elements inline",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"x"},
						Attributes: map[string]Value{
							"tags": listVal(strVal("docker"), strVal("linux")),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "x" {
  tags = ["docker", "linux"]
}`,
		},
		{
			name: "list with 3 elements multi-line with trailing comma",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"x"},
						Attributes: map[string]Value{
							"tags": listVal(strVal("docker"), strVal("linux"), strVal("amd64")),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "x" {
  tags = [
    "docker",
    "linux",
    "amd64",
  ]
}`,
		},
		{
			name: "multiple top-level blocks separated by exactly one blank line",
			config: &Config{
				Blocks: []Block{
					{Type: "egg", Labels: []string{"a"}, Attributes: map[string]Value{}, Blocks: nil},
					{Type: "egg", Labels: []string{"b"}, Attributes: map[string]Value{}, Blocks: nil},
					{Type: "egg", Labels: []string{"c"}, Attributes: map[string]Value{}, Blocks: nil},
				},
			},
			want: `egg "a" {
}

egg "b" {
}

egg "c" {
}`,
		},
		{
			name: "no trailing newline at EOF",
			config: &Config{
				Blocks: []Block{
					{Type: "egg", Labels: []string{"x"}, Attributes: map[string]Value{}, Blocks: nil},
				},
			},
			want: `egg "x" {
}`,
		},
		{
			name: "2-space indentation per nesting level",
			config: &Config{
				Blocks: []Block{
					{
						Type:       "egg",
						Labels:     []string{"x"},
						Attributes: map[string]Value{},
						Blocks: []Block{
							{
								Type:       "runner",
								Labels:     nil,
								Attributes: map[string]Value{},
								Blocks: []Block{
									{
										Type:       "limits",
										Labels:     nil,
										Attributes: map[string]Value{"cpu": numVal(4)},
										Blocks:     nil,
									},
								},
							},
						},
					},
				},
			},
			want: `egg "x" {
  runner {
    limits {
      cpu = 4
    }
  }
}`,
		},
		{
			name: "opening brace on same line closing brace on own line",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"x"},
						Attributes: map[string]Value{
							"type": strVal("vm"),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "x" {
  type = "vm"
}`,
		},
		{
			name: "empty config produces empty string",
			config: &Config{
				Blocks: nil,
			},
			want: "",
		},
		{
			name: "bool and number values",
			config: &Config{
				Blocks: []Block{
					{
						Type:   "egg",
						Labels: []string{"x"},
						Attributes: map[string]Value{
							"enabled":  boolVal(true),
							"disabled": boolVal(false),
							"count":    numVal(42),
							"ratio":    numVal(1.5),
						},
						Blocks: nil,
					},
				},
			},
			want: `egg "x" {
  count = 42
  disabled = false
  enabled = true
  ratio = 1.5
}`,
		},
		{
			name: "block with no labels",
			config: &Config{
				Blocks: []Block{
					{
						Type:       "uglyfox",
						Labels:     nil,
						Attributes: map[string]Value{"max_age": numVal(3600)},
						Blocks:     nil,
					},
				},
			},
			want: `uglyfox {
  max_age = 3600
}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := f.Format(tc.config)
			if got != tc.want {
				t.Errorf("Format() mismatch\ngot:\n%s\n\nwant:\n%s", got, tc.want)
			}
		})
	}
}

// TestFormatterNoTrailingNewline explicitly verifies the last byte is not '\n'.
func TestFormatterNoTrailingNewline(t *testing.T) {
	f := NewFormatter()
	config := &Config{
		Blocks: []Block{
			{Type: "egg", Labels: []string{"x"}, Attributes: map[string]Value{}, Blocks: nil},
		},
	}
	got := f.Format(config)
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Errorf("Format() output ends with trailing newline")
	}
}

// TestFormatterExactlyOneBlankLineBetweenBlocks verifies the separator is "\n\n" (not more).
func TestFormatterExactlyOneBlankLineBetweenBlocks(t *testing.T) {
	f := NewFormatter()
	config := &Config{
		Blocks: []Block{
			{Type: "egg", Labels: []string{"a"}, Attributes: map[string]Value{}, Blocks: nil},
			{Type: "egg", Labels: []string{"b"}, Attributes: map[string]Value{}, Blocks: nil},
		},
	}
	got := f.Format(config)
	// The separator between the two blocks must be exactly one blank line ("\n\n").
	// Three or more consecutive newlines would mean extra blank lines.
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("Format() contains more than one blank line between blocks:\n%s", got)
	}
}
