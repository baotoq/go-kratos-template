package biz

import (
	"net/http"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrCoffeeNotFound_is_a_404(t *testing.T) {
	// Arrange / Act
	err := kratoserrors.FromError(ErrCoffeeNotFound)

	// Assert
	assert.Equal(t, int32(http.StatusNotFound), err.Code)
	assert.Equal(t, "COFFEE_NOT_FOUND", err.Reason)
}
