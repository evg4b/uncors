package validators_test

//import (
//	"testing"
//
//	"github.com/evg4b/uncors/internal/config"
//	"github.com/stretchr/testify/assert"
//)
//
//func TestValidate(t *testing.T) {
//	tests := []struct {
//		name     string
//		config   *config.UncorsConfig
//		expected string
//	}{
//		{
//			name: "invalid http-port",
//			config: &config.UncorsConfig{
//				Mappings: config.Mappings{},
//			},
//			expected: "Key: 'UncorsConfig.HTTPPort' Error:Field validators for 'HTTPPort' failed on the 'required' tag",
//		},
//	}
//	for _, testCase := range tests {
//		t.Run(testCase.name, func(t *testing.T) {
//			err := config.Validate(testCase.config)
//
//			assert.EqualError(t, err, testCase.expected)
//		})
//	}
//}
