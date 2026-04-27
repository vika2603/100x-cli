// Package session resolves a per-invocation execution context: a configured
// futures.Client plus the profile metadata used to build it.
//
// Both the cobra PersistentPreRunE path and the shell-completion path go
// through Load. The two callers differ only in their HTTP timeout and in
// whether they propagate or swallow errors; both differences are expressed
// as parameters and return values, never as duplicate pipelines.
package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/vika2603/100x-cli/api/futures"
	"github.com/vika2603/100x-cli/internal/config"
	"github.com/vika2603/100x-cli/internal/credential"
)

// Session is a fully-resolved execution context for one CLI invocation.
//
// For Public sessions, Profile is nil and ProfileName is empty.
type Session struct {
	Client      *futures.Client
	Profile     *config.Profile
	ProfileName string
	Endpoint    string
}

// LoadOptions controls how Load resolves the session.
type LoadOptions struct {
	// RequestedProfile is the explicit --profile flag value. Empty falls back
	// to E100X_PROFILE / Config.Default per config.Resolve.
	RequestedProfile string

	// Timeout is a backstop on the underlying http.Client. The real
	// per-command budget comes from the caller's ctx.
	Timeout time.Duration

	// Public, when true, builds an unsigned client suitable for public market
	// endpoints. Profile resolution and credential loading are skipped.
	Public bool
}

const httpClientBackstop = 5 * time.Minute

// Load resolves the session for one CLI invocation.
//
// For Public:true, Load skips credential I/O and builds an unsigned client
// against the configured endpoint; it may still return config.ErrNoEndpoint.
// For private sessions, Load may return config.ErrNoProfile (no profile
// configured), a "profile not found" error, config.ErrNoEndpoint, or a
// wrapped credential-load error; callers decide whether to surface or
// swallow them.
func Load(opts LoadOptions) (Session, error) {
	httpClient := &http.Client{Timeout: httpClientBackstop}

	if opts.Public {
		endpoint, err := config.Endpoint()
		if err != nil {
			return Session{}, err
		}
		return Session{
			Client: futures.New(futures.Options{
				Endpoint:   endpoint,
				HTTPClient: httpClient,
			}),
			Endpoint: endpoint,
		}, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return Session{}, err
	}
	name, p, err := config.Resolve(cfg, opts.RequestedProfile)
	if err != nil {
		return Session{}, err
	}
	endpoint, err := config.Endpoint()
	if err != nil {
		return Session{}, err
	}
	env, err := credential.LoadSecret(p.ClientID)
	if err != nil {
		return Session{}, fmt.Errorf("load credentials for profile %q: %w", name, err)
	}
	return Session{
		Client: futures.New(futures.Options{
			Endpoint:   endpoint,
			ClientID:   p.ClientID,
			ClientKey:  env.Secret,
			HTTPClient: httpClient,
		}),
		Profile:     p,
		ProfileName: name,
		Endpoint:    endpoint,
	}, nil
}
