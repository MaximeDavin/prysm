package config

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestConfigApply(t *testing.T) {
	c := NewConfig()
	require.Equal(t, false, c.UseQuic)

	setQuic := func(cfg *Config) error {
		cfg.UseQuic = true
		return nil
	}
	err := c.Apply(nil, setQuic)
	require.NoError(t, err)
	require.Equal(t, true, c.UseQuic)

	errorOption := func(cfg *Config) error {
		return fmt.Errorf("wrong options")
	}
	err = c.Apply(errorOption)
	require.ErrorContains(t, "wrong options", err)

}
