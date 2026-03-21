package parser

import (
	"fmt"
	"sort"
	"strings"
)

// Formatter renders a Config AST into canonical .fly format.
type Formatter struct{}

// NewFormatter creates a new Formatter instance.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Format renders the full config as a canonical string.
// Top-level blocks are separated by exactly one blank line; no trailing newline.
func (f *Formatter) Format(config *Config) string {
	if len(config.Blocks) == 0 {
		return ""
	}

	parts := make([]string, 0, len(config.Blocks))
	for i := range config.Blocks {
		parts = append(parts, f.formatBlock(&config.Blocks[i], 0))
	}

	return strings.Join(parts, "\n\n")
}

// formatBlock renders a single block recursively.
// indent is the current nesting level (0 = top-level).
func (f *Formatter) formatBlock(block *Block, indent int) string {
	prefix := strings.Repeat("  ", indent)
	var sb strings.Builder

	// Header: type + labels + opening brace
	sb.WriteString(prefix)
	sb.WriteString(block.Type)
	for _, label := range block.Labels {
		sb.WriteString(fmt.Sprintf(" %q", label))
	}
	sb.WriteString(" {")

	// Collect attribute keys and sort alphabetically
	attrKeys := make([]string, 0, len(block.Attributes))
	for k := range block.Attributes {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)

	// Write attributes
	childPrefix := strings.Repeat("  ", indent+1)
	for _, key := range attrKeys {
		val := block.Attributes[key]
		sb.WriteString("\n")
		sb.WriteString(childPrefix)
		sb.WriteString(key)
		sb.WriteString(" = ")
		sb.WriteString(f.formatValue(&val, indent+1))
	}

	// Write nested blocks in source order
	for i := range block.Blocks {
		sb.WriteString("\n")
		sb.WriteString(f.formatBlock(&block.Blocks[i], indent+1))
	}

	// Closing brace at block's own indent level
	sb.WriteString("\n")
	sb.WriteString(prefix)
	sb.WriteString("}")

	return sb.String()
}

// formatValue renders a Value to its canonical string representation.
// indent is the indent level of the containing block (used for multi-line lists).
func (f *Formatter) formatValue(value *Value, indent int) string {
	switch value.Type {
	case StringType:
		return fmt.Sprintf("%q", value.Raw.(string))

	case NumberType:
		num := value.Raw.(float64)
		// Render integers without decimal point
		if num == float64(int64(num)) {
			return fmt.Sprintf("%d", int64(num))
		}
		return fmt.Sprintf("%g", num)

	case BoolType:
		if value.Raw.(bool) {
			return "true"
		}
		return "false"

	case ListType:
		list := value.Raw.([]Value)
		if len(list) <= 2 {
			// Inline format
			items := make([]string, 0, len(list))
			for i := range list {
				items = append(items, f.formatValue(&list[i], indent))
			}
			return "[" + strings.Join(items, ", ") + "]"
		}
		// Multi-line format: each element on its own line with trailing comma
		childPrefix := strings.Repeat("  ", indent+1)
		closingPrefix := strings.Repeat("  ", indent)
		var sb strings.Builder
		sb.WriteString("[")
		for i := range list {
			sb.WriteString("\n")
			sb.WriteString(childPrefix)
			sb.WriteString(f.formatValue(&list[i], indent+1))
			sb.WriteString(",")
		}
		sb.WriteString("\n")
		sb.WriteString(closingPrefix)
		sb.WriteString("]")
		return sb.String()

	case MapType:
		m := value.Raw.(map[string]Value)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		childPrefix := strings.Repeat("  ", indent+1)
		closingPrefix := strings.Repeat("  ", indent)
		var sb strings.Builder
		sb.WriteString("{")
		for _, k := range keys {
			v := m[k]
			sb.WriteString("\n")
			sb.WriteString(childPrefix)
			sb.WriteString(k)
			sb.WriteString(" = ")
			sb.WriteString(f.formatValue(&v, indent+1))
		}
		sb.WriteString("\n")
		sb.WriteString(closingPrefix)
		sb.WriteString("}")
		return sb.String()

	default:
		return fmt.Sprintf("%v", value.Raw)
	}
}
