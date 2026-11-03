package http

import (
	"net/http"

	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/fransalwan/secure-payment-gateway/pkg/response"
)

type AITelemetryHandler struct {
	outboxRepo *postgresrepo.OutboxRepository
}

func NewAITelemetryHandler(outboxRepo *postgresrepo.OutboxRepository) *AITelemetryHandler {
	return &AITelemetryHandler{outboxRepo: outboxRepo}
}

// GetTelemetrySummary returns high-level AI Observability & FinOps metrics
func (h *AITelemetryHandler) GetTelemetrySummary(w http.ResponseWriter, r *http.Request) {
	if h.outboxRepo == nil {
		response.InternalServerError(w, "Outbox repository not initialized")
		return
	}

	summary, err := h.outboxRepo.GetAITelemetrySummary(r.Context())
	if err != nil {
		response.InternalServerError(w, "Failed to fetch AI telemetry: "+err.Error())
		return
	}

	response.OK(w, "AI Telemetry and FinOps metrics retrieved", summary)
}

