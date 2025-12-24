package parser

import (
	"fmt"
	"strings"
)

// Position represents a location in the source file
type Position struct {
	Line   int
	Column int
	File   string
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
}

// Node is the base interface for all AST nodes
type Node interface {
	Pos() Position
	String() string
}

// Config represents the root of a .fly configuration file
type Config struct {
	Position Position
	Blocks   []Block
}

func (c *Config) Pos() Position {
	return c.Position
}

func (c *Config) String() string {
	var sb strings.Builder
	for _, block := range c.Blocks {
		sb.WriteString(block.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// Block represents a configuration block (e.g., egg, job, uglyfox)
type Block struct {
	Position   Position
	Type       string           // "egg", "job", "uglyfox", etc.
	Labels     []string         // Block labels (e.g., ["my-app"] for egg "my-app")
	Attributes map[string]Value // Direct attributes
	Blocks     []Block          // Nested blocks
}

func (b *Block) Pos() Position {
	return b.Position
}

func (b *Block) String() string {
	var sb strings.Builder
	sb.WriteString(b.Type)
	for _, label := range b.Labels {
		sb.WriteString(fmt.Sprintf(" %q", label))
	}
	sb.WriteString(" {\n")

	// Write attributes
	for key, val := range b.Attributes {
		sb.WriteString(fmt.Sprintf("  %s = %s\n", key, val.String()))
	}

	// Write nested blocks
	for _, nested := range b.Blocks {
		lines := strings.Split(nested.String(), "\n")
		for _, line := range lines {
			if line != "" {
				sb.WriteString("  ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("}")
	return sb.String()
}

// GetAttribute retrieves an attribute by name
func (b *Block) GetAttribute(name string) (Value, bool) {
	val, ok := b.Attributes[name]
	return val, ok
}

// GetBlock retrieves the first nested block of a given type
func (b *Block) GetBlock(blockType string) (*Block, bool) {
	for i := range b.Blocks {
		if b.Blocks[i].Type == blockType {
			return &b.Blocks[i], true
		}
	}
	return nil, false
}

// GetBlocks retrieves all nested blocks of a given type
func (b *Block) GetBlocks(blockType string) []Block {
	var result []Block
	for _, nested := range b.Blocks {
		if nested.Type == blockType {
			result = append(result, nested)
		}
	}
	return result
}

// ValueType represents the type of a value
type ValueType int

const (
	StringType ValueType = iota
	NumberType
	BoolType
	ListType
	MapType
)

func (vt ValueType) String() string {
	switch vt {
	case StringType:
		return "string"
	case NumberType:
		return "number"
	case BoolType:
		return "bool"
	case ListType:
		return "list"
	case MapType:
		return "map"
	default:
		return "unknown"
	}
}

// Value represents a value in the configuration
type Value struct {
	Position Position
	Type     ValueType
	Raw      interface{} // Actual value: string, float64, bool, []Value, map[string]Value
}

func (v *Value) Pos() Position {
	return v.Position
}

func (v *Value) String() string {
	switch v.Type {
	case StringType:
		return fmt.Sprintf("%q", v.Raw.(string))
	case NumberType:
		return fmt.Sprintf("%v", v.Raw)
	case BoolType:
		return fmt.Sprintf("%v", v.Raw)
	case ListType:
		list := v.Raw.([]Value)
		var items []string
		for _, item := range list {
			items = append(items, item.String())
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	case MapType:
		m := v.Raw.(map[string]Value)
		var pairs []string
		for k, val := range m {
			pairs = append(pairs, fmt.Sprintf("%s = %s", k, val.String()))
		}
		return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
	default:
		return fmt.Sprintf("%v", v.Raw)
	}
}

// AsString returns the value as a string
func (v *Value) AsString() (string, error) {
	if v.Type != StringType {
		return "", fmt.Errorf("expected string, got %s at %s", v.Type, v.Position)
	}
	return v.Raw.(string), nil
}

// AsNumber returns the value as a float64
func (v *Value) AsNumber() (float64, error) {
	if v.Type != NumberType {
		return 0, fmt.Errorf("expected number, got %s at %s", v.Type, v.Position)
	}
	return v.Raw.(float64), nil
}

// AsInt returns the value as an int
func (v *Value) AsInt() (int, error) {
	num, err := v.AsNumber()
	if err != nil {
		return 0, err
	}
	return int(num), nil
}

// AsBool returns the value as a bool
func (v *Value) AsBool() (bool, error) {
	if v.Type != BoolType {
		return false, fmt.Errorf("expected bool, got %s at %s", v.Type, v.Position)
	}
	return v.Raw.(bool), nil
}

// AsList returns the value as a list
func (v *Value) AsList() ([]Value, error) {
	if v.Type != ListType {
		return nil, fmt.Errorf("expected list, got %s at %s", v.Type, v.Position)
	}
	return v.Raw.([]Value), nil
}

// AsMap returns the value as a map
func (v *Value) AsMap() (map[string]Value, error) {
	if v.Type != MapType {
		return nil, fmt.Errorf("expected map, got %s at %s", v.Type, v.Position)
	}
	return v.Raw.(map[string]Value), nil
}

// Equals checks if two values are equal
func (v *Value) Equals(other *Value) bool {
	if v.Type != other.Type {
		return false
	}

	switch v.Type {
	case StringType:
		return v.Raw.(string) == other.Raw.(string)
	case NumberType:
		return v.Raw.(float64) == other.Raw.(float64)
	case BoolType:
		return v.Raw.(bool) == other.Raw.(bool)
	case ListType:
		vList := v.Raw.([]Value)
		oList := other.Raw.([]Value)
		if len(vList) != len(oList) {
			return false
		}
		for i := range vList {
			if !vList[i].Equals(&oList[i]) {
				return false
			}
		}
		return true
	case MapType:
		vMap := v.Raw.(map[string]Value)
		oMap := other.Raw.(map[string]Value)
		if len(vMap) != len(oMap) {
			return false
		}
		for k, vVal := range vMap {
			oVal, ok := oMap[k]
			if !ok || !vVal.Equals(&oVal) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
