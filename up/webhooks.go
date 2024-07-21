package up

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const upAuthenticityHeader = "X-Up-Authenticity-Signature"

var (
	webhookRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "up_bank_webhook_requests",
		Help: "Up bank webhook requests with full information labels",
	}, []string{"webhook_id", "event_type"})
	webhookInflights = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "up_bank_webhook_incoming_inflights",
		Help: "Number of inflight incoming webhook requests",
	})
	transactionCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "up_bank_transaction_count",
		Help: "Up bank transaction count processed by the webhook handler",
	}, []string{"account_id", "status"})
	transactionAmount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "up_bank_transaction_amount",
		Help: "Up bank transaction amount in base units processed by the webhook handler. The amount of money in the smallest denomination for the currency, as a 64-bit integer.  For example, for an Australian dollar value of $10.56, this field will be `1056`.",
	}, []string{"account_id", "status"})
)

type UpWebhookHandler struct {
	secretKey []byte

	c ClientWithResponsesInterface
}

func NewUpWebhookHandler(secretKey []byte) *UpWebhookHandler {
	return &UpWebhookHandler{secretKey: secretKey}
}

func (h *UpWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	webhookInflights.Inc()
	defer webhookInflights.Dec()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	authenticitySignature, err := hex.DecodeString(r.Header.Get(upAuthenticityHeader))
	if err != nil || len(authenticitySignature) == 0 {
		http.Error(w, fmt.Sprintf("%s header missing or invalid", upAuthenticityHeader), http.StatusUnauthorized)
		return
	}

	if !h.authenticate(body, authenticitySignature) {
		http.Error(w, "signature verification failed", http.StatusUnauthorized)
		return
	}

	data := &WebhookEventCallback{}
	if err := json.Unmarshal(body, data); err != nil {
		http.Error(w, fmt.Sprintf("failed to unmarshal request body: %v", err), http.StatusBadRequest)
	}

	h.handleWebhookRequest(ctx, &data.Data)
}

func (h *UpWebhookHandler) authenticate(body []byte, authenticitySignature []byte) bool {
	mac := hmac.New(sha256.New, h.secretKey)
	if _, err := mac.Write(body); err != nil {
		// hash.Hash docs say this never returns an error.
		log.Printf("hash.Hash.Writer returned an unexpected error: %v\n", err)
		return false
	}
	expected := mac.Sum(nil)
	return hmac.Equal(expected, []byte(authenticitySignature))
}

func (h *UpWebhookHandler) handleWebhookRequest(ctx context.Context, r *WebhookEventResource) error {
	eventType := fmt.Sprintf("%s", r.Attributes.EventType)
	webhookID := r.Relationships.Webhook.Data.Id
	webhookRequests.With(prometheus.Labels{"event_type": eventType, "webhook_id": webhookID})

	if t := r.Relationships.Transaction; t != nil {
		return h.handleTransaction(ctx, t.Data.Id)
	}
	return nil
}

func (h *UpWebhookHandler) handleTransaction(ctx context.Context, id string) error {
	t, err := h.c.GetTransactionsIdWithResponse(ctx, id)
	if err != nil {
		return err
	}

	accountID := t.JSON200.Data.Relationships.Account.Data.Id
	status := fmt.Sprintf("%s", t.JSON200.Data.Attributes.Status)
	amount := float64(t.JSON200.Data.Attributes.Amount.ValueInBaseUnits)

	transactionLabels := prometheus.Labels{
		"account_id":    accountID,
		"status":        status,
		"currency_code": t.JSON200.Data.Attributes.Amount.CurrencyCode,
	}
	transactionCount.With(transactionLabels).Inc()
	transactionAmount.With(transactionLabels).Add(amount)
	return nil
}
