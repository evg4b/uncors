package helpers_test

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
	"testing"
)

type myService struct {
	service string
}

func TestApplyOptions(t *testing.T) {
	t.Run("ApplyOptions sets service value", func(t *testing.T) {
		const testValue = "TestValue"

		service := &myService{}
		options := []func(*myService){
			func(s *myService) {
				s.service = testValue
			},
		}

		result := helpers.ApplyOptions(service, options)

		assert.Equal(t, service, result, "The same service should be returned")
		assert.Equal(t, testValue, result.service, "Service option should be applied")
	})

	t.Run("ApplyOptions handles empty options", func(t *testing.T) {
		service := &myService{}

		result := helpers.ApplyOptions(service, nil)

		assert.Equal(t, service, result, "The same service should be returned")
		assert.Equal(t, "", result.service, "Service value should not be applied")
	})
}
