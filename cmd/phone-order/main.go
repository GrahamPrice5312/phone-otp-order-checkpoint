package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"example.com/phone-otp-orders/internal/commerce"
)

type server struct {
	service *commerce.Service
}

type sendRequest struct {
	Phone          string `json:"phone"`
	WidgetRecordID string `json:"widget_record_id"`
	CaptchaToken   string `json:"captcha_token"`
	RequestID      string `json:"request_id"`
}

type confirmRequest struct {
	Phone     string            `json:"phone"`
	Code      string            `json:"code"`
	RequestID string            `json:"request_id"`
	Checkout  commerce.Checkout `json:"checkout"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := commerce.NewOTPClient(key)
	service := commerce.NewService(client)
	s := &server{service: service}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", s.sendCode)
	mux.HandleFunc("POST /checkout/confirm", s.confirmCheckout)
	mux.HandleFunc("GET /orders/{order_id}", s.getOrder)

	log.Println("phone-order listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) sendCode(w http.ResponseWriter, r *http.Request) {
	var input sendRequest
	if err := decode(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.service.StartLogin(r.Context(), input.Phone, input.WidgetRecordID, input.CaptchaToken, clientIP(r), input.RequestID); err != nil {
		writeJSON(w, commerce.ClientStatus(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "code_sent"})
}

func (s *server) confirmCheckout(w http.ResponseWriter, r *http.Request) {
	var input confirmRequest
	if err := decode(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	update, err := s.service.ConfirmCheckout(r.Context(), input.Phone, input.Code, input.RequestID, input.Checkout)
	if err != nil {
		writeJSON(w, commerce.ClientStatus(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, update)
}

func (s *server) getOrder(w http.ResponseWriter, r *http.Request) {
	order, ok := s.service.Order(r.PathValue("order_id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid JSON request")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}
