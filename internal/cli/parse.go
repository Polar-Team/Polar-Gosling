package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/polar-gosling/gosling/internal/parser"
	"github.com/spf13/cobra"
)

var (
	parseType string
)

// parseCmd represents the parse command
var parseCmd = &cobra.Command{
	Use:   "parse [file]",
	Short: "Parse .fly configuration file and output JSON",
	Long: `Parse a .fly configuration file and output the parsed structure as JSON.

The JSON output contains the complete parsed configuration structure with snake_case field names
for Python compatibility. This command is used by MotherGoose backend to parse .fly files.

Example:
  gosling parse Eggs/my-app/config.fly --type egg
  gosling parse Jobs/rotate-secrets.fly --type job
  gosling parse UF/config.fly --type uglyfox`,
	Args: cobra.ExactArgs(1),
	RunE: runParse,
}

func init() {
	rootCmd.AddCommand(parseCmd)
	parseCmd.Flags().StringVarP(&parseType, "type", "t", "", "Configuration type (egg, job, uglyfox, eggsbucket)")
}

func runParse(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Parse the .fly file
	config, err := parser.ParseAndValidate(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		return fmt.Errorf("parse failed")
	}

	// Validate type if specified
	if parseType != "" {
		if err := validateConfigType(config, parseType); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return fmt.Errorf("type validation failed")
		}
	}

	// Convert to JSON-serializable structure with snake_case
	jsonData := configToJSON(config)

	// Output JSON to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonData); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		return fmt.Errorf("json encoding failed")
	}

	return nil
}

func validateConfigType(config *parser.Config, expectedType string) error {
	if len(config.Blocks) == 0 {
		return fmt.Errorf("configuration file is empty")
	}

	// Check if all blocks match the expected type
	for _, block := range config.Blocks {
		if block.Type != expectedType {
			return fmt.Errorf("expected block type %q, got %q", expectedType, block.Type)
		}
	}

	return nil
}

// configToJSON converts a Config to a JSON-serializable map with snake_case field names
func configToJSON(config *parser.Config) map[string]interface{} {
	blocks := make([]map[string]interface{}, 0, len(config.Blocks))
	for _, block := range config.Blocks {
		blocks = append(blocks, blockToJSON(&block))
	}

	return map[string]interface{}{
		"blocks": blocks,
	}
}

// blockToJSON converts a Block to a JSON-serializable map with snake_case field names
func blockToJSON(block *parser.Block) map[string]interface{} {
	result := map[string]interface{}{
		"type":   block.Type,
		"labels": block.Labels,
	}

	// Convert attributes
	if len(block.Attributes) > 0 {
		attrs := make(map[string]interface{})
		for key, val := range block.Attributes {
			attrs[key] = valueToJSON(&val)
		}
		result["attributes"] = attrs
	}

	// Convert nested blocks
	if len(block.Blocks) > 0 {
		nestedBlocks := make([]map[string]interface{}, 0, len(block.Blocks))
		for i := range block.Blocks {
			nestedBlocks = append(nestedBlocks, blockToJSON(&block.Blocks[i]))
		}
		result["blocks"] = nestedBlocks
	}

	return result
}

// valueToJSON converts a Value to a JSON-serializable interface{}
func valueToJSON(val *parser.Value) interface{} {
	switch val.Type {
	case parser.StringType:
		return val.Raw.(string)
	case parser.NumberType:
		return val.Raw.(float64)
	case parser.BoolType:
		return val.Raw.(bool)
	case parser.ListType:
		list := val.Raw.([]parser.Value)
		result := make([]interface{}, 0, len(list))
		for i := range list {
			result = append(result, valueToJSON(&list[i]))
		}
		return result
	case parser.MapType:
		m := val.Raw.(map[string]parser.Value)
		result := make(map[string]interface{})
		for k, v := range m {
			result[k] = valueToJSON(&v)
		}
		return result
	default:
		return val.Raw
	}
}
