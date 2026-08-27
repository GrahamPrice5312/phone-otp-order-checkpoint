package commerce

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type PhoneVerifier interface {
	VerifyCaptcha(context.Context, string, string, string) error
	SendCode(context.Context, string, string) error
	VerifyCode(context.Context, string, string, string) error
}

type Checkout struct {
	OrderID string `json:"order_id"`
	SKU     string `json:"sku"`
	Units   int    `json:"units"`
}

type OrderUpdate struct {
	OrderID        string `json:"order_id"`
	Phone          string `json:"phone"`
	Checkout       string `json:"checkout"`
	Fulfillment    string `json:"fulfillment"`
	Receipt        string `json:"receipt"`
	CustomerUpdate string `json:"customer_update"`
}

type Service struct {
	otp    PhoneVerifier
	mu     sync.RWMutex
	orders map[string]OrderUpdate
}

func NewService(otp PhoneVerifier) *Service {
	return &Service{otp: otp, orders: make(map[string]OrderUpdate)}
}

func (s *Service) StartLogin(ctx context.Context, phone, widgetRecordID, captchaToken, ip, requestID string) error {
	if phone == "" || widgetRecordID == "" || captchaToken == "" || requestID == "" {
		return errors.New("phone, widget_record_id, captcha_token, and request_id are required")
	}
	if err := s.otp.VerifyCaptcha(ctx, widgetRecordID, captchaToken, ip); err != nil {
		return err
	}
	return s.otp.SendCode(ctx, phone, requestID)
}

func (s *Service) ConfirmCheckout(ctx context.Context, phone, code, requestID string, checkout Checkout) (OrderUpdate, error) {
	if phone == "" || code == "" || requestID == "" || checkout.OrderID == "" || checkout.SKU == "" || checkout.Units < 1 {
		return OrderUpdate{}, errors.New("valid phone, code, request_id, and checkout are required")
	}
	if err := s.otp.VerifyCode(ctx, phone, code, requestID); err != nil {
		return OrderUpdate{}, err
	}

	update := OrderUpdate{
		OrderID:        checkout.OrderID,
		Phone:          phone,
		Checkout:       "confirmed",
		Fulfillment:    "queued",
		Receipt:        fmt.Sprintf("receipt:%s", checkout.OrderID),
		CustomerUpdate: "order_confirmed",
	}
	s.mu.Lock()
	s.orders[checkout.OrderID] = update
	s.mu.Unlock()
	return update, nil
}

func (s *Service) Order(orderID string) (OrderUpdate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[orderID]
	return order, ok
}
