package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"l0/internal/service"
	"l0/pkg/er"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

type Response struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		orderUID := vars["order_uid"]
		if orderUID == "" {
			writeJSONResponse(w, http.StatusBadRequest, Response{
				Status: "error",
				Msg:    "order_uid is required",
			})
			return
		}
		order, err := h.svc.GetOrderResponse(r.Context(), orderUID)
		if err != nil {
			if errors.Is(err, er.ErrOrderNotFound) {
				zap.S().Infof("failed to get order: %v", err)
				writeJSONResponse(w, http.StatusNotFound, Response{
					Status: "error",
					Msg:    "order not found",
					Data:   nil,
				})
				return
			}
			zap.S().Infof("failed to get order: %v", err)
			writeJSONResponse(w, http.StatusInternalServerError, Response{
				Status: "error",
				Msg:    "internal server error",
				Data:   nil,
			})
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{
			Status: "ok",
			Msg:    "",
			Data:   order,
		})
	}
}
