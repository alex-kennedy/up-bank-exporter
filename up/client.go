package up

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/errgroup"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config=config.yaml ../api/v1/openapi.json

const UP_API_ADDRESS = "https://api.up.com.au/api/v1"

// Page size for paginated requests.
var pageSize int = 100

var (
	accountsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "up_bank_accounts_count",
		Help: "Count of Up bank accounts",
	})
	accountBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "up_bank_account_balance",
		Help: "Up bank account balance in base currency units (e.g. cents)",
	}, []string{"id", "display_name", "account_type", "ownership_type", "currency_code"})
	webhooksCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "up_bank_webhooks_count",
		Help: "Count of configured Up webhooks",
	})
)

type UpMetricsClient struct {
	client ClientWithResponsesInterface
}

func NewUpMetricsClient(bearerToken string) (*UpMetricsClient, error) {
	auth, err := securityprovider.NewSecurityProviderBearerToken(bearerToken)
	if err != nil {
		return nil, err
	}
	client, err := NewClientWithResponses(UP_API_ADDRESS,
		WithRequestEditorFn(auth.Intercept),
		WithHTTPClient(NewPromHTTPClient()),
	)
	if err != nil {
		return nil, err
	}
	return &UpMetricsClient{client: client}, err
}

func (u *UpMetricsClient) UpdateMetrics(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return u.UpdateAccountsMetrics(ctx)
	})
	g.Go(func() error {
		return u.UpdateWebhookMetrics(ctx)
	})
	return g.Wait()
}

func (u *UpMetricsClient) UpdateAccountsMetrics(ctx context.Context) error {
	var accounts []AccountResource

	accountsResp, err := u.client.GetAccountsWithResponse(ctx, &GetAccountsParams{PageSize: &pageSize})
	if err != nil {
		return err
	}
	accounts = append(accounts, accountsResp.JSON200.Data...)
	for accountsResp.JSON200.Links.Next != nil {
		accountsResp, err = u.client.GetAccountsWithResponse(ctx, nil, func(_ context.Context, req *http.Request) error {
			return updateRequestWithPageToken(*accountsResp.JSON200.Links.Next, req)
		})
		if err != nil {
			return err
		}
		accounts = append(accounts, accountsResp.JSON200.Data...)
	}

	accountsCount.Set(float64(len(accounts)))
	for _, account := range accounts {
		accountBalance.With(prometheus.Labels{
			"id":             account.Id,
			"display_name":   account.Attributes.DisplayName,
			"account_type":   fmt.Sprintf("%s", account.Attributes.AccountType),
			"ownership_type": fmt.Sprintf("%s", account.Attributes.OwnershipType),
			"currency_code":  account.Attributes.Balance.CurrencyCode,
		}).Set(float64(account.Attributes.Balance.ValueInBaseUnits))
	}

	return nil
}

func (u *UpMetricsClient) UpdateWebhookMetrics(ctx context.Context) error {
	var webhooks []WebhookResource

	webhooksResp, err := u.client.GetWebhooksWithResponse(ctx, &GetWebhooksParams{PageSize: &pageSize})
	if err != nil {
		return err
	}
	webhooks = append(webhooks, webhooksResp.JSON200.Data...)
	for webhooksResp.JSON200.Links.Next != nil {
		webhooksResp, err = u.client.GetWebhooksWithResponse(ctx, nil, func(_ context.Context, req *http.Request) error {
			return updateRequestWithPageToken(*webhooksResp.JSON200.Links.Next, req)
		})
		if err != nil {
			return err
		}
		webhooks = append(webhooks, webhooksResp.JSON200.Data...)
	}

	webhooksCount.Set(float64(len(webhooks)))
	return nil
}

func updateRequestWithPageToken(path string, req *http.Request) error {
	tokenUrl, err := url.Parse(path)
	if err != nil {
		return err
	}
	req.URL = tokenUrl
	return nil
}
