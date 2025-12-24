package parser

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Parser parses .fly configuration files
type Parser struct {
	parser *hclparse.Parser
}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile parses a .fly file and returns the AST
func (p *Parser) ParseFile(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return p.Parse(content, filename)
}

// Parse parses .fly content and returns the AST
func (p *Parser) Parse(content []byte, filename string) (*Config, error) {
	file, diags := p.parser.ParseHCL(content, filename)
	if diags.HasErrors() {
		return nil, p.formatDiagnostics(diags)
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type")
	}

	config := &Config{
		Position: Position{
			File:   filename,
			Line:   1,
			Column: 1,
		},
		Blocks: make([]Block, 0),
	}

	// Parse top-level blocks
	for _, hclBlock := range body.Blocks {
		block, err := p.parseBlock(hclBlock, filename)
		if err != nil {
			return nil, err
		}
		config.Blocks = append(config.Blocks, *block)
	}

	return config, nil
}

// parseBlock converts an HCL block to our AST Block
func (p *Parser) parseBlock(hclBlock *hclsyntax.Block, filename string) (*Block, error) {
	block := &Block{
		Position: Position{
			File:   filename,
			Line:   hclBlock.TypeRange.Start.Line,
			Column: hclBlock.TypeRange.Start.Column,
		},
		Type:       hclBlock.Type,
		Labels:     hclBlock.Labels,
		Attributes: make(map[string]Value),
		Blocks:     make([]Block, 0),
	}

	// Parse attributes
	for name, attr := range hclBlock.Body.Attributes {
		val, err := p.parseExpression(attr.Expr, filename)
		if err != nil {
			return nil, fmt.Errorf("error parsing attribute %s: %w", name, err)
		}
		block.Attributes[name] = *val
	}

	// Parse nested blocks
	for _, nestedHCL := range hclBlock.Body.Blocks {
		nested, err := p.parseBlock(nestedHCL, filename)
		if err != nil {
			return nil, err
		}
		block.Blocks = append(block.Blocks, *nested)
	}

	return block, nil
}

// parseExpression converts an HCL expression to our Value type
func (p *Parser) parseExpression(expr hclsyntax.Expression, filename string) (*Value, error) {
	pos := Position{
		File:   filename,
		Line:   expr.Range().Start.Line,
		Column: expr.Range().Start.Column,
	}

	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		return p.parseLiteralValue(e, pos)

	case *hclsyntax.TemplateExpr:
		// Handle string templates (including simple strings)
		if len(e.Parts) == 1 {
			if lit, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				return p.parseLiteralValue(lit, pos)
			}
		}
		// For complex templates, we'll need to evaluate them
		// For now, return an error for unsupported template expressions
		return nil, fmt.Errorf("complex template expressions not yet supported at %s", pos)

	case *hclsyntax.TupleConsExpr:
		// Parse list/array
		list := make([]Value, 0, len(e.Exprs))
		for _, itemExpr := range e.Exprs {
			item, err := p.parseExpression(itemExpr, filename)
			if err != nil {
				return nil, err
			}
			list = append(list, *item)
		}
		return &Value{
			Position: pos,
			Type:     ListType,
			Raw:      list,
		}, nil

	case *hclsyntax.ObjectConsExpr:
		// Parse map/object
		m := make(map[string]Value)
		for _, item := range e.Items {
			// Get the key
			keyExpr, ok := item.KeyExpr.(*hclsyntax.ObjectConsKeyExpr)
			if !ok {
				return nil, fmt.Errorf("unsupported map key type at %s", pos)
			}

			// For now, we only support simple string keys
			key := ""
			if len(keyExpr.Wrapped.(*hclsyntax.TemplateExpr).Parts) == 1 {
				if lit, ok := keyExpr.Wrapped.(*hclsyntax.TemplateExpr).Parts[0].(*hclsyntax.LiteralValueExpr); ok {
					key = lit.Val.AsString()
				}
			}

			if key == "" {
				return nil, fmt.Errorf("invalid map key at %s", pos)
			}

			// Get the value
			val, err := p.parseExpression(item.ValueExpr, filename)
			if err != nil {
				return nil, err
			}
			m[key] = *val
		}
		return &Value{
			Position: pos,
			Type:     MapType,
			Raw:      m,
		}, nil

	case *hclsyntax.ScopeTraversalExpr:
		// Variable reference - for now, return as string representation
		// In a full implementation, we'd resolve these during evaluation
		return &Value{
			Position: pos,
			Type:     StringType,
			Raw:      fmt.Sprintf("${%s}", e.Traversal.RootName()),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported expression type %T at %s", expr, pos)
	}
}

// parseLiteralValue converts an HCL literal value to our Value type
func (p *Parser) parseLiteralValue(lit *hclsyntax.LiteralValueExpr, pos Position) (*Value, error) {
	ctyVal := lit.Val
	ctyType := ctyVal.Type()

	// Check for string type
	if ctyType.Equals(cty.String) {
		return &Value{
			Position: pos,
			Type:     StringType,
			Raw:      ctyVal.AsString(),
		}, nil
	}

	// Check for number type
	if ctyType.Equals(cty.Number) {
		num, _ := ctyVal.AsBigFloat().Float64()
		return &Value{
			Position: pos,
			Type:     NumberType,
			Raw:      num,
		}, nil
	}

	// Check for bool type
	if ctyType.Equals(cty.Bool) {
		return &Value{
			Position: pos,
			Type:     BoolType,
			Raw:      ctyVal.True(),
		}, nil
	}

	return nil, fmt.Errorf("unsupported literal type %s at %s", ctyType.FriendlyName(), pos)
}

// formatDiagnostics formats HCL diagnostics into a readable error message
func (p *Parser) formatDiagnostics(diags hcl.Diagnostics) error {
	var messages []string
	for _, diag := range diags {
		msg := fmt.Sprintf("%s: %s", diag.Subject, diag.Detail)
		if diag.Context != nil {
			msg = fmt.Sprintf("%s (context: %s)", msg, *diag.Context)
		}
		messages = append(messages, msg)
	}
	return fmt.Errorf("parse errors:\n%s", joinMessages(messages))
}

func joinMessages(messages []string) string {
	result := ""
	for i, msg := range messages {
		if i > 0 {
			result += "\n"
		}
		result += fmt.Sprintf("  - %s", msg)
	}
	return result
}
