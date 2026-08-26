package unleash_test

import (
	"context"
	"testing"

	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/infras/unleash"

	"github.com/stretchr/testify/assert"
)

func TestNew_ReturnsNoop_WhenURLEmpty(t *testing.T) {
	cfg := &config.Config{}
	cfg.Unleash.URL = ""

	ff, err := unleash.New(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, ff)
	result := ff.IsEnabled(context.Background(), "any-flag")
	assert.False(t, result)
}

func TestNew_ReturnsNoop_WhenNoUnleashConfig(t *testing.T) {
	cfg := &config.Config{}

	ff, err := unleash.New(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, ff)
	result := ff.IsEnabled(context.Background(), "any-flag")
	assert.False(t, result)
}

func TestNew_ReturnsFeatureFlag_WhenURLProvided(t *testing.T) {
	cfg := &config.Config{}
	cfg.Unleash.URL = "http://localhost:4242/api"
	cfg.Unleash.Secret = "secret-token"
	cfg.Unleash.AppName = "oil"

	ff, err := unleash.New(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, ff)
	result := ff.IsEnabled(context.Background(), "any-flag")
	assert.False(t, result)
}
