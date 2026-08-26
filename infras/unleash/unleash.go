// Package unleash provides feature flag integration via the Unleash SDK.
package unleash

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/savioruz/oil/config"

	"github.com/Unleash/unleash-go-sdk/v6"
	sdkctx "github.com/Unleash/unleash-go-sdk/v6/context"
)

const defaultRefreshInterval = 15 * time.Second

// FeatureFlag is the interface for checking feature flag states.
type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

type noopFeatureFlag struct{}

// IsEnabled implements FeatureFlag. Always returns false for noop.
func (n *noopFeatureFlag) IsEnabled(_ context.Context, _ string) bool {
	return false
}

// Client implements FeatureFlag using the Unleash SDK.
type Client struct {
	mu     sync.RWMutex
	client *unleash.Client
}

// New returns a FeatureFlag implementation. Returns a noop (always false)
// when Unleash URL is not configured or on any client creation error.
func New(cfg *config.Config) (FeatureFlag, error) {
	if cfg.Unleash.URL == "" {
		return &noopFeatureFlag{}, nil
	}

	appName := cfg.Unleash.AppName
	if appName == "" {
		appName = cfg.App.Name
	}

	instanceID := cfg.Unleash.InstanceID

	headers := make(http.Header)
	headers.Set("Authorization", cfg.Unleash.Secret)

	client, err := unleash.NewClient(
		unleash.WithUrl(cfg.Unleash.URL),
		unleash.WithAppName(appName),
		unleash.WithInstanceId(instanceID),
		unleash.WithRefreshInterval(defaultRefreshInterval),
		unleash.WithDisableMetrics(false),
		unleash.WithCustomHeaders(headers),
	)
	if err != nil {
		return &noopFeatureFlag{}, nil
	}

	return &Client{client: client}, nil
}

// IsEnabled checks if a feature flag is enabled. Returns false on any error (fail open).
func (u *Client) IsEnabled(_ context.Context, flag string) bool {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.client == nil {
		return false
	}

	enabled := u.client.IsEnabled(flag, unleash.FeatureOptions{
		Ctx: sdkctx.Context{},
	})

	return enabled
}
