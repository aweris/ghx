package actions

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		value    yaml.Marshaler
		expected interface{}
	}{
		{
			name:     "bool with value",
			value:    &Bool{Value: true},
			expected: true,
		},
		{
			name:     "bool with expression",
			value:    &Bool{Expression: &String{Value: "${{ example }}"}},
			expected: "${{ example }}",
		},
		{
			name:     "int with value",
			value:    &Int{Value: 42},
			expected: 42,
		},
		{
			name:     "int with expression",
			value:    &Int{Expression: &String{Value: "${{ example }}"}},
			expected: "${{ example }}",
		},
		{
			name:     "float with value",
			value:    &Float{Value: 3.14},
			expected: 3.14,
		},
		{
			name:     "float with expression",
			value:    &Float{Expression: &String{Value: "${{ example }}"}},
			expected: "${{ example }}",
		},
		{
			name:     "string with value",
			value:    &String{Value: "foobar"},
			expected: "foobar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.value.MarshalYAML()

			if err != nil {
				t.Errorf("Expected no error in MarshalYAML, but got %s", err.Error())
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %s, but got %s", tt.expected, result)
			}
		})
	}
}

func TestUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		node     *yaml.Node
		expected yaml.Unmarshaler
	}{
		{
			name:     "bool with value",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
			expected: &Bool{Value: true},
		},
		{
			name:     "bool with expression",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "${{ example }}"},
			expected: &Bool{Expression: &String{Value: "${{ example }}"}},
		},
		{
			name:     "int with value",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "42"},
			expected: &Int{Value: 42},
		},
		{
			name:     "int with expression",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "${{ example }}"},
			expected: &Int{Expression: &String{Value: "${{ example }}"}},
		},
		{
			name:     "float with value",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "3.14"},
			expected: &Float{Value: 3.14},
		},
		{
			name:     "float with expression",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "${{ example }}"},
			expected: &Float{Expression: &String{Value: "${{ example }}"}},
		},
		{
			name:     "string with value",
			node:     &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "foobar"},
			expected: &String{Value: "foobar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reflect.New(reflect.TypeOf(tt.expected).Elem()).Interface().(yaml.Unmarshaler)
			err := result.UnmarshalYAML(tt.node)

			if err != nil {
				t.Errorf("Expected no error in UnmarshalYAML, but got %s", err.Error())
			}

			if reflect.TypeOf(result) != reflect.TypeOf(tt.expected) {
				t.Errorf("Expected %s in UnmarshalYAML, but got %s", reflect.TypeOf(tt.expected), reflect.TypeOf(result))
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %s, but got %s", tt.expected, result)
			}
		})
	}
}
