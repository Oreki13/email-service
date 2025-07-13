package repository

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLTemplateRepository_parseVariables(t *testing.T) {
	// Create repository instance for testing
	repo := &SQLTemplateRepository{db: &sql.DB{}}

	tests := []struct {
		name           string
		input          []byte
		expectedResult map[string]interface{}
		expectError    bool
	}{
		{
			name:           "Empty JSON",
			input:          []byte{},
			expectedResult: nil,
			expectError:    false,
		},
		{
			name:           "Valid object JSON",
			input:          []byte(`{"Name": "string", "AppName": "string"}`),
			expectedResult: map[string]interface{}{"Name": "string", "AppName": "string"},
			expectError:    false,
		},
		{
			name:           "Valid array JSON (legacy format)",
			input:          []byte(`["Name", "AppName", "Email"]`),
			expectedResult: map[string]interface{}{"Name": "", "AppName": "", "Email": ""},
			expectError:    false,
		},
		{
			name:           "Empty object JSON",
			input:          []byte(`{}`),
			expectedResult: map[string]interface{}{},
			expectError:    false,
		},
		{
			name:           "Empty array JSON",
			input:          []byte(`[]`),
			expectedResult: map[string]interface{}{},
			expectError:    false,
		},
		{
			name:        "Invalid JSON",
			input:       []byte(`invalid json`),
			expectError: true,
		},
		{
			name:        "Unsupported JSON type (number)",
			input:       []byte(`123`),
			expectError: true,
		},
		{
			name:        "Unsupported JSON type (string)",
			input:       []byte(`"test"`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.parseVariables(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestSQLTemplateRepository_parseVariables_ComplexObjects(t *testing.T) {
	repo := &SQLTemplateRepository{db: &sql.DB{}}

	tests := []struct {
		name           string
		input          []byte
		expectedResult map[string]interface{}
	}{
		{
			name:  "Complex object with nested types",
			input: []byte(`{"Name": "John", "Age": 30, "IsActive": true, "Metadata": {"Department": "IT"}}`),
			expectedResult: map[string]interface{}{
				"Name":     "John",
				"Age":      float64(30), // JSON numbers are parsed as float64
				"IsActive": true,
				"Metadata": map[string]interface{}{"Department": "IT"},
			},
		},
		{
			name:  "Array with mixed variable names",
			input: []byte(`["firstName", "lastName", "email_address", "user.id"]`),
			expectedResult: map[string]interface{}{
				"firstName":     "",
				"lastName":      "",
				"email_address": "",
				"user.id":       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.parseVariables(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
