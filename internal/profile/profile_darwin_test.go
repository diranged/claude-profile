//go:build darwin

package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurrentUser(t *testing.T) {
	u := currentUser()
	assert.NotEmpty(t, u)
}
