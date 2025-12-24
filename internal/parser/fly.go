package parser

import "fmt"

// ParseAndValidate parses a .fly file and validates it
func ParseAndValidate(filename string) (*Config, error) {
	parser := NewParser()
	config, err := parser.ParseFile(filename)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if !result.IsValid() {
		return nil, fmt.Errorf("validation error: %w", result)
	}

	return config, nil
}

// ParseAndValidateContent parses .fly content and validates it
func ParseAndValidateContent(content []byte, filename string) (*Config, error) {
	parser := NewParser()
	config, err := parser.Parse(content, filename)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	validator := NewValidator(config)
	result := validator.Validate()
	if !result.IsValid() {
		return nil, fmt.Errorf("validation error: %w", result)
	}

	return config, nil
}
