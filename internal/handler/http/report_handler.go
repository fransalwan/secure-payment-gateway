package http

import (
	"encoding/json"
	"net/http"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/fransalwan/secure-payment-gateway/internal/service"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
)

type ReportHandler struct {
	reportService *service.ReportAIService
}

func NewReportHandler(reportService *service.ReportAIService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) AskAIReport(w http.ResponseWriter, r *http.Request) {
	var req domain.AskAIReportRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON request body", "INVALID_JSON", err.Error())
		return
	}

	result, err := h.reportService.AskAIReport(r.Context(), &req)
	if err != nil {
		response.BadRequest(w, "Failed to generate report", "REPORT_ERROR", err.Error())
		return
	}

	response.OK(w, "AI Natural Language report generated successfully", result)
}
