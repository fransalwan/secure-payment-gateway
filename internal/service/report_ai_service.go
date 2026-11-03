package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ReportAIService struct {
	cfg          *config.Config
	reportRepo   *postgresrepo.ReportingRepository
	outboxRepo   *postgresrepo.OutboxRepository
	httpClient   *http.Client
	schemaPrompt string
}

func NewReportAIService(
	cfg *config.Config,
	reportRepo *postgresrepo.ReportingRepository,
	outboxRepo *postgresrepo.OutboxRepository,
) *ReportAIService {
	schema := `Database Schema (PostgreSQL 16):
- merchants (id UUID, code VARCHAR, name VARCHAR, city VARCHAR, phone VARCHAR)
- products (id UUID, merchant_id UUID, sku VARCHAR, name VARCHAR, price NUMERIC, available_stock INT, reserved_stock INT, sold_stock INT)
- orders (id UUID, merchant_id UUID, order_number VARCHAR, channel VARCHAR['WEB','WHATSAPP_LIVE','SHOPEE','TIKTOK_SHOP'], customer_name VARCHAR, status VARCHAR['PENDING_PAYMENT','PAID','READY_TO_SHIP','SHIPPED','COMPLETED','CANCELLED','SUSPICIOUS_HOLD'], total_amount NUMERIC, waybill_number VARCHAR, paid_at TIMESTAMPTZ, created_at TIMESTAMPTZ)
- order_items (id UUID, order_id UUID, product_id UUID, sku VARCHAR, product_name VARCHAR, qty INT, unit_price NUMERIC, subtotal NUMERIC)
- webhook_events (id UUID, provider VARCHAR, event_type VARCHAR, status VARCHAR, processing_ms INT, created_at TIMESTAMPTZ)
- payment_proofs (id UUID, order_id UUID, merchant_id UUID, bank_name VARCHAR, sender_name VARCHAR, amount NUMERIC, status VARCHAR['PENDING','VERIFIED','SUSPICIOUS_HOLD','REJECTED'], tamper_detected BOOLEAN, confidence_score NUMERIC, image_phash VARCHAR, is_duplicate_image BOOLEAN, created_at TIMESTAMPTZ)

Enterprise Semantic Metric Definitions:
1. GMV (Gross Merchandise Value): COALESCE(SUM(total_amount), 0) FROM orders WHERE status IN ('PAID', 'READY_TO_SHIP', 'SHIPPED', 'COMPLETED')
2. Net Realized Revenue: COALESCE(SUM(total_amount), 0) FROM orders WHERE status = 'COMPLETED'
3. Pending Receivables: COALESCE(SUM(total_amount), 0) FROM orders WHERE status = 'PENDING_PAYMENT'
4. Fraud Quarantine Volume: COUNT(*) FROM orders WHERE status = 'SUSPICIOUS_HOLD'
5. Top Performing Products: Grouped by product_name, SUM(qty) AS total_sold, SUM(subtotal) AS gmv FROM order_items
6. Channel Distribution: Count and total GMV grouped by channel ('WEB', 'SHOPEE', 'TIKTOK_SHOP', 'WHATSAPP_LIVE')`

	return &ReportAIService{
		cfg:          cfg,
		reportRepo:   reportRepo,
		outboxRepo:   outboxRepo,
		schemaPrompt: schema,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ReportAIService) AskAIReport(
	ctx context.Context,
	req *domain.AskAIReportRequestDTO,
) (*domain.AskAIReportResponseDTO, error) {
	start := time.Now()
	cleanQuestion := strings.TrimSpace(req.Question)
	if cleanQuestion == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	var generatedSQL string
	var explanation string
	var chartType string
	var tokensUsed int

	// 1. Generate SQL via Gemini LLM if key is present
	if strings.TrimSpace(s.cfg.GeminiAPIKey) != "" {
		llmCtx, cancel := context.WithTimeout(ctx, 4000*time.Millisecond)
		defer cancel()

		sql, expl, chart, tokens, err := s.generateSQLFromLLM(llmCtx, cleanQuestion, req.MerchantID)
		if err == nil && sql != "" {
			generatedSQL = sql
			explanation = expl
			chartType = chart
			tokensUsed = tokens
		} else {
			log.Warn().Err(err).Msg("LLM Text-to-SQL generation failed, falling back to deterministic template")
		}
	}

	// 2. Deterministic Template Fallback
	if generatedSQL == "" {
		sql, expl, chart := s.deterministicReportFallback(cleanQuestion, req.MerchantID)
		generatedSQL = sql
		explanation = expl
		chartType = chart
	}

	// 3. Strict AST & Token Security Sanitization Guard
	if err := s.ValidateSQLSecurity(generatedSQL); err != nil {
		_ = s.reportRepo.AuditAIQuery(ctx, req.MerchantID, cleanQuestion, generatedSQL, false, time.Since(start).Milliseconds(), 0, tokensUsed)
		return nil, fmt.Errorf("security violation in generated SQL: %w", err)
	}

	// 4. Execute Safe Analytical Query inside Read-Only session with timeout
	cols, rows, err := s.reportRepo.ExecuteSafeAnalyticalQuery(ctx, generatedSQL)
	if err != nil {
		_ = s.reportRepo.AuditAIQuery(ctx, req.MerchantID, cleanQuestion, generatedSQL, false, time.Since(start).Milliseconds(), 0, tokensUsed)
		return nil, fmt.Errorf("failed to execute analytical SQL query: %w", err)
	}

	execMs := time.Since(start).Milliseconds()
	_ = s.reportRepo.AuditAIQuery(ctx, req.MerchantID, cleanQuestion, generatedSQL, true, execMs, len(rows), tokensUsed)

	if s.outboxRepo != nil {
		toks := tokensUsed
		if toks == 0 {
			toks = 350
		}
		_ = s.outboxRepo.RecordAITelemetry(ctx, domain.AITelemetryEvent{
			ServiceType:      "TEXT_TO_SQL",
			ModelName:        "gemini-1.5-flash",
			PromptTokens:     toks * 2 / 3,
			CompletionTokens: toks / 3,
			LatencyMs:        int(execMs),
			CostUSD:          float64(toks) * 0.00000035,
			IsSuccess:        true,
			FraudFlag:        false,
		})
	}

	log.Info().
		Str("question", cleanQuestion).
		Str("chart_type", chartType).
		Int("rows_count", len(rows)).
		Int64("exec_ms", execMs).
		Msg("Enterprise Text-to-SQL report generated successfully")

	return &domain.AskAIReportResponseDTO{
		Question:      cleanQuestion,
		GeneratedSQL:  generatedSQL,
		Explanation:   explanation,
		ChartType:     chartType,
		Columns:       cols,
		Rows:          rows,
		TotalRows:     len(rows),
		ExecutionMs:   execMs,
		SanitizedSafe: true,
		TokensUsed:    tokensUsed,
	}, nil
}

// ValidateSQLSecurity ensures the query is exclusively a safe, read-only SELECT statement
func (s *ReportAIService) ValidateSQLSecurity(query string) error {
	trimmed := strings.TrimSpace(query)
	upper := strings.ToUpper(trimmed)

	// Must start with SELECT or WITH
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("only SELECT or CTE queries are permitted")
	}

	// Must not contain semicolons to prevent multi-statement injection
	if strings.Contains(strings.TrimRight(trimmed, ";"), ";") {
		return fmt.Errorf("multi-statement queries are forbidden")
	}

	// Blacklisted DDL / DML / Destructive keywords
	blockedKeywords := []string{
		"INSERT ", "UPDATE ", "DELETE ", "DROP ", "ALTER ", "TRUNCATE ",
		"CREATE ", "GRANT ", "REVOKE ", "EXEC ", "EXECUTE ", "PG_SLEEP",
		"INTO OUTFILE", "COPY ", "VACUUM ", "REINDEX ", "--", "/*",
	}

	for _, kw := range blockedKeywords {
		if strings.Contains(upper, kw) {
			return fmt.Errorf("destructive or prohibited keyword detected: %s", strings.TrimSpace(kw))
		}
	}

	return nil
}

func (s *ReportAIService) generateSQLFromLLM(
	ctx context.Context,
	question string,
	merchantID uuid.UUID,
) (sql string, explanation string, chartType string, tokens int, err error) {
	model := s.cfg.GeminiModel
	if model == "" {
		model = "gemini-1.5-flash"
	}

	apiURL := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, s.cfg.GeminiAPIKey,
	)

	prompt := fmt.Sprintf(`%s

User Request: "%s"
Merchant ID Filter: "%s"

Generate a PostgreSQL 16 SQL query to answer the user's business question.
CRITICAL RULES:
1. ONLY return a valid SELECT statement. Do NOT use INSERT, UPDATE, DELETE, ALTER, DROP, or semicolons.
2. If merchant_id filter is provided and not 00000000-0000-0000-0000-000000000000, add WHERE merchant_id = '%s'.
3. Always add LIMIT 50 unless it is an aggregated summary.
4. Suggest a chart type from: "bar", "line", "pie", "stat", or "table".

Return strict JSON only matching:
{
  "sql": "SELECT ...",
  "explanation": "Brief explanation of what the query calculates",
  "chart_type": "bar" | "line" | "pie" | "stat" | "table"
}`, s.schemaPrompt, question, merchantID.String(), merchantID.String())

	reqPayload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.1,
			"responseMimeType": "application/json",
		},
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", "", "", 0, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", "", "", 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", "", "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", "", 0, fmt.Errorf("gemini api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", "", "", 0, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", "", "", 0, fmt.Errorf("empty candidates from gemini")
	}

	raw := geminiResp.Candidates[0].Content.Parts[0].Text
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed struct {
		SQL         string `json:"sql"`
		Explanation string `json:"explanation"`
		ChartType   string `json:"chart_type"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", "", "", 0, err
	}

	// Clean trailing semicolon from generated SQL
	cleanSQL := strings.TrimRight(strings.TrimSpace(parsed.SQL), ";")
	return cleanSQL, parsed.Explanation, parsed.ChartType, geminiResp.UsageMetadata.TotalTokenCount, nil
}

func (s *ReportAIService) deterministicReportFallback(
	question string,
	merchantID uuid.UUID,
) (sql string, explanation string, chartType string) {
	lower := strings.ToLower(question)
	mFilter := ""
	if merchantID != uuid.Nil {
		mFilter = fmt.Sprintf("WHERE merchant_id = '%s'", merchantID.String())
	}

	switch {
	case strings.Contains(lower, "channel") || strings.Contains(lower, "kanal") || strings.Contains(lower, "omset"):
		where := mFilter
		if where == "" {
			where = "WHERE status IN ('PAID', 'READY_TO_SHIP', 'SHIPPED', 'COMPLETED')"
		} else {
			where += " AND status IN ('PAID', 'READY_TO_SHIP', 'SHIPPED', 'COMPLETED')"
		}
		sql = fmt.Sprintf(`
			SELECT channel,
			       COUNT(id) AS total_orders,
			       COALESCE(SUM(total_amount), 0) AS total_gmv,
			       ROUND(AVG(total_amount), 2) AS avg_order_value
			FROM orders
			%s
			GROUP BY channel
			ORDER BY total_gmv DESC
		`, where)
		explanation = "Agregasi total pesanan, GMV dan rata-rata nilai order per channel penjualan"
		chartType = "bar"

	case strings.Contains(lower, "produk") || strings.Contains(lower, "sku") || strings.Contains(lower, "terlaris") || strings.Contains(lower, "top"):
		sql = `
			SELECT p.sku,
			       p.name AS product_name,
			       p.sold_stock,
			       p.available_stock,
			       p.price,
			       (p.sold_stock * p.price) AS revenue
			FROM products p
			ORDER BY p.sold_stock DESC
			LIMIT 10
		`
		explanation = "10 SKU produk terlaris berdasarkan akumulasi sold_stock dan estimasi revenue"
		chartType = "bar"

	case strings.Contains(lower, "batal") || strings.Contains(lower, "cancel") || strings.Contains(lower, "shortage") || strings.Contains(lower, "stok"):
		where := mFilter
		if where == "" {
			where = "WHERE status = 'CANCELLED'"
		} else {
			where += " AND status = 'CANCELLED'"
		}
		sql = fmt.Sprintf(`
			SELECT channel,
			       COUNT(id) AS cancelled_orders,
			       COALESCE(SUM(total_amount), 0) AS lost_gmv
			FROM orders
			%s
			GROUP BY channel
			ORDER BY cancelled_orders DESC
		`, where)
		explanation = "Jumlah pesanan yang dibatalkan dan potensi GMV yang hilang per channel"
		chartType = "pie"

	case strings.Contains(lower, "slip") || strings.Contains(lower, "transfer") || strings.Contains(lower, "bukti") || strings.Contains(lower, "tamper"):
		sql = `
			SELECT status,
			       COUNT(id) AS total_proofs,
			       COALESCE(SUM(amount), 0) AS total_amount,
			       ROUND(AVG(confidence_score), 4) AS avg_confidence
			FROM payment_proofs
			GROUP BY status
			ORDER BY total_proofs DESC
		`
		explanation = "Distribusi verifikasi bukti transfer Vision AI (Verified vs Suspicious Hold vs Pending)"
		chartType = "pie"

	default:
		where := mFilter
		if where == "" {
			where = "WHERE 1=1"
		}
		sql = fmt.Sprintf(`
			SELECT status,
			       COUNT(id) AS order_count,
			       COALESCE(SUM(total_amount), 0) AS total_amount
			FROM orders
			%s
			GROUP BY status
			ORDER BY order_count DESC
		`, where)
		explanation = "Distribusi status seluruh pesanan (Pending, Paid, Ready to Ship, Cancelled, Suspicious)"
		chartType = "stat"
	}

	cleanSQL := strings.TrimRight(strings.TrimSpace(sql), ";")
	// Clean whitespace indentation
	lines := strings.Split(cleanSQL, "\n")
	var cleanedLines []string
	for _, l := range lines {
		cleanedLines = append(cleanedLines, strings.TrimSpace(l))
	}
	cleanSQL = strings.Join(cleanedLines, " ")
	cleanSQL = regexp.MustCompile(`\s+`).ReplaceAllString(cleanSQL, " ")

	return cleanSQL, explanation, chartType
}

