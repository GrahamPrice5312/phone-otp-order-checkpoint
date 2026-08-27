package commerce

import (
	"context"
	"errors"
	"testing"
)

type stubVerifier struct {
	verifyErr error
	verified  bool
}

func (s *stubVerifier) VerifyCaptcha(context.Context, string, string, string) error { return nil }
func (s *stubVerifier) SendCode(context.Context, string, string) error              { return nil }
func (s *stubVerifier) VerifyCode(context.Context, string, string, string) error {
	s.verified = true
	return s.verifyErr
}

func TestConfirmCheckoutDecision(t *testing.T) {
	tests := []struct {
		name       string
		verifyErr  error
		wantStored bool
		wantState  string
	}{
		{name: "verified phone queues fulfillment", wantStored: true, wantState: "queued"},
		{name: "rejected code leaves no order", verifyErr: errors.New("code rejected"), wantStored: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			verifier := &stubVerifier{verifyErr: tc.verifyErr}
			service := NewService(verifier)
			checkout := Checkout{OrderID: "ord-2048", SKU: "cable-usbc", Units: 2}

			got, err := service.ConfirmCheckout(context.Background(), "+15551234567", "123456", "req-test", checkout)
			if (err == nil) != tc.wantStored {
				t.Fatalf("ConfirmCheckout error = %v, wantStored = %v", err, tc.wantStored)
			}
			if !verifier.verified {
				t.Fatal("checkout decision skipped phone verification")
			}
			stored, ok := service.Order(checkout.OrderID)
			if ok != tc.wantStored {
				t.Fatalf("Order stored = %v, want %v", ok, tc.wantStored)
			}
			if tc.wantStored && (got.Fulfillment != tc.wantState || stored.Receipt != "receipt:ord-2048") {
				t.Fatalf("unexpected order update: %+v", got)
			}
		})
	}
}
