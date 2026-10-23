package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFractionalTokenCharge(t *testing.T) {
	rules := []priceRule{
		{ID: "input", PriceType: "input", PriceUnit: "1k_tokens", Price: 0.025, ProviderPrice: 0.0002, PriceMetadata: "{}"},
		{ID: "output", PriceType: "output", PriceUnit: "1k_tokens", Price: 0.15, ProviderPrice: 0.0012, PriceMetadata: "{}"},
	}
	charge, err := chargeForUsage(selectedModelRoute{}, nil, upstreamChatResult{PromptTokens: 1000, CompletionTokens: 100}, rules, 100)
	if err != nil {
		t.Fatal(err)
	}
	if charge.CustomerCharge != 0.04 || charge.ProviderCost != 0.032 {
		t.Fatalf("unexpected fractional charge: %+v", charge)
	}
}

func TestPriceAlternativesAndProfileOverrides(t *testing.T) {
	rules := []priceRule{
		{ID: "base", PriceType: "output", PriceUnit: "image", Price: 100, PriceMetadata: "{}"},
		{ID: "small", PriceType: "output", PriceUnit: "image", Price: 5, ProfileMatched: true, PriceMetadata: `{"resolution":"512x512"}`},
		{ID: "large", PriceType: "output", PriceUnit: "image", Price: 10, ProfileMatched: true, PriceMetadata: `{"resolution":"1024x1024"}`},
	}
	selected, err := selectPriceRules(rules, response{"resolution": "512x512"})
	if err != nil || len(selected) != 1 || selected[0].ID != "small" {
		t.Fatalf("selected=%+v err=%v", selected, err)
	}
	if _, err := selectPriceRules(rules, nil); err == nil {
		t.Fatal("ambiguous alternatives must not be summed")
	}
	if _, err := selectPriceRules(rules[1:], response{"resolution": "unsupported"}); err == nil {
		t.Fatal("unpriced resolution must be rejected")
	}
}

func TestMissingPricesRejectRequest(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select pr.id, pr.pricing_variant").WithArgs("unpriced", "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "variant", "type", "unit", "price", "provider_price", "metadata", "profile"}))
	if _, err := app.calculateRequestCharge(context.Background(), selectedModelRoute{ModelSlug: "unpriced"}, nil, upstreamChatResult{}); err == nil {
		t.Fatal("missing prices must not default to one credit")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSettlementRollsBackWhenUsagePersistenceFails(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select paid_credits, promotional_credits").WithArgs("wallet-1").
		WillReturnRows(sqlmock.NewRows([]string{"paid", "promo"}).AddRow(9.975, 0))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", "capture_req_1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_usage_events").WillReturnError(errors.New("storage unavailable"))
	mock.ExpectRollback()
	err := app.captureProjectCredits(context.Background(), "req_1", creditReservation{WalletID: "wallet-1", PaidCredits: 0.025}, 0.025, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(context.Background(), "insert into user_usage_events")
		return err
	})
	if err == nil {
		t.Fatal("failed usage persistence must roll back settlement")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBillingQuantitiesRejectInvalidValues(t *testing.T) {
	for _, parameters := range []response{
		{"output_count": -1}, {"output_count": 5}, {"duration_seconds": 1.5}, {"max_tokens": "invalid"},
	} {
		if err := validateBillingParameters(parameters); err == nil {
			t.Fatalf("accepted invalid quantities: %+v", parameters)
		}
	}
}

func TestSplitFractionalAssetChargesPreservesTotal(t *testing.T) {
	var total float64
	for i := int64(0); i < 3; i++ {
		total += splitCredits(0.025, 3, i)
	}
	if roundCredits(total) != 0.025 {
		t.Fatalf("split changed charge: %g", total)
	}
}

func TestEmptyUserUsageDoesNotSubstituteDemoUser(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select id from sys_users").WithArgs("user-empty").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-empty"))
	mock.ExpectQuery("select to_char.*ue.created_at").WithArgs("user-empty", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"date", "slug", "name", "modality", "input", "output", "charge"}))
	mock.ExpectQuery("select to_char.*created_at").WithArgs("user-empty", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"date", "credits", "cents", "currency", "status"}))
	recorder := httptest.NewRecorder()
	app.userCreditUsage(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user-credit-usage?user_id=user-empty", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"user_id":"user-empty"`) || !strings.Contains(recorder.Body.String(), `"usage":[]`) {
		t.Fatalf("unexpected usage response: %d %s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
