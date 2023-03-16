package runner

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_expBackoff(t *testing.T) {
	req := require.New(t)

	req.Equal(expBackoff(), 30)
	req.Equal(expBackoff(), 30*2)
	req.Equal(expBackoff(), 30*2*2)
	req.Equal(expBackoff(), 30*2*2*2)
	req.Equal(expBackoff(), 30*2*2*2*2)
	resetBackoff()
	req.Equal(expBackoff(), 30)
	req.Equal(expBackoff(), 30*2)
	req.Equal(expBackoff(), 30*2*2)
	req.Equal(expBackoff(), 30*2*2*2)
	req.Equal(expBackoff(), 30*2*2*2*2)
}
