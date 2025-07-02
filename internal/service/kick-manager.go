package service

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/pkg/assert"
	sharedService "github.com/arnokay/arnobot-shared/service"
	"github.com/arnokay/arnobot-shared/trace"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/scorfly/gokick"
)

// TODO: right now there is no cleanup for clients
type KickManager struct {
	logger       *slog.Logger
	clientID     string
	clientSecret string

	appClient *gokick.Client

	clients map[string]*gokick.Client
	mu      sync.RWMutex

	cache      jetstream.KeyValue
	authModule *sharedService.AuthModule
}

func NewKickManager(
	cache jetstream.KeyValue,
	authModule *sharedService.AuthModule,
	clientID, clientSecret string,
) *KickManager {
	logger := applog.NewServiceLogger("kick-manager")

	appClient, err := gokick.NewClient(&gokick.ClientOptions{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	assert.NoError(err, "gokick client needs to be initialized")

	token, err := appClient.GetAppAccessToken(context.TODO())
	assert.NoError(err, "cannot get access tokens for app client")

	appClient.SetAppAccessToken(token.AccessToken)

	return &KickManager{
		logger:       logger,
		clientID:     clientID,
		clientSecret: clientSecret,
		appClient:    appClient,
		clients:      make(map[string]*gokick.Client),
		cache:        cache,
		authModule:   authModule,
	}
}

func (hm *KickManager) GetApp(ctx context.Context) *gokick.Client {
	return hm.appClient
}

func (hm *KickManager) GetByID(ctx context.Context, kickID string) (*gokick.Client, error) {
	hm.mu.RLock()
	client, exists := hm.clients[kickID]
	hm.mu.RUnlock()

	if exists {
		return client, nil
	}

	return nil, apperror.New(apperror.CodeNotFound, "gokick client is not found", nil)
}

// TODO: remove later, i need this to find a bug
type customRoundTripper struct {
	logger *slog.Logger
}

func (l *customRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	l.logRequest(req)

	return http.DefaultTransport.RoundTrip(req)
}

func (l *customRoundTripper) logRequest(req *http.Request) {
	l.logger.Debug("[REQUEST]", "method", req.Method, "url", req.URL.String())

	for name, values := range req.Header {
		for _, value := range values {
			if name == "Authorization" || name == "Cookie" {
				l.logger.Debug("[REQUEST HEADER]", "name", name, "value", "[REDACTED]")
			} else {
				l.logger.Debug("[REQUEST HEADER]", "name", name, "value", value)
			}
		}
	}

	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			// Restore the body for the actual request
			req.Body = io.NopCloser(bytes.NewReader(body))
			l.logger.Debug("[REQUEST BODY]", "body", string(body))
		}
	}
}

func (hm *KickManager) GetByProvider(ctx context.Context, provider data.AuthProvider) *gokick.Client {
	client, err := hm.GetByID(ctx, provider.ProviderUserID)

	if err == nil {
		return client
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if client, exists := hm.clients[provider.ProviderUserID]; exists {
		return client
	}

	client, _ = gokick.NewClient(&gokick.ClientOptions{
		HTTPClient: &http.Client{
			Transport: &customRoundTripper{logger: hm.logger},
		},
		ClientID:         hm.clientID,
		ClientSecret:     hm.clientSecret,
		UserAccessToken:  provider.AccessToken,
		UserRefreshToken: provider.RefreshToken,
	})

	client.OnUserAccessTokenRefreshed(func(newAccessToken, newRefreshToken string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ctx = trace.Context(ctx, trace.New())
		defer cancel()
		// COMBAK: maybe set ttl?
		hm.cache.Put(
			ctx,
			"hm.art."+provider.Provider+"."+provider.ProviderUserID,
			bytes.Join([][]byte{[]byte(newAccessToken), []byte(newRefreshToken)}, []byte("...")),
		)
		hm.logger.InfoContext(ctx, "token refreshed", "providerUserID", provider.ProviderUserID)
		err := hm.authModule.AuthProviderUpdateTokens(ctx, data.AuthProviderUpdateTokens{
			ID:           provider.ID,
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		})
		if err != nil {
			hm.logger.ErrorContext(ctx, "failed to update tokens", "providerID", provider.ID, "providerUserID", provider.ProviderUserID)
		}
	})

	hm.clients[provider.ProviderUserID] = client

	return client
}
