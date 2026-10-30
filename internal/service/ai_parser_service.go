package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	postgresrepo "github.com/fransalwan/secure-payment-gateway/internal/repository/postgres"
	"github.com/google/uuid"
)

type AIOrderParserService struct {
	omsRepo    *postgresrepo.OMSRepository
	cfg        *config.Config
	httpClient *http.Client
}

func NewAIOrderParserService(omsRepo *postgresrepo.OMSRepository, cfg *config.Config) *AIOrderParserService {
	return &AIOrderParserService{
		omsRepo: omsRepo,
		cfg:     cfg,
		httpClient: &http.Client{
			Timeout: 2600 * time.Millisecond, // Enforce < 3s AC strictly
		},
	}
}

type llmExtractedDraft struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	ShippingAddress string `json:"shipping_address"`
	CourierCode     string `json:"courier_code"`
	Items           []struct {
		SKU         string `json:"sku"`
		ProductName string `json:"product_name"`
		Qty         int    `json:"qty"`
	} `json:"items"`
}

func (s *AIOrderParserService) ParseSocialCommerceChat(
	ctx context.Context,
	merchantID uuid.UUID,
	rawText string,
) (*domain.AIParseResponseDTO, error) {
	start := time.Now()
	cleanText := strings.TrimSpace(rawText)
	if cleanText == "" {
		return nil, fmt.Errorf("chat text cannot be empty")
	}

	products, err := s.omsRepo.ListProductsByMerchant(ctx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch merchant SKU catalog: %w", err)
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("merchant has no active SKU catalog")
	}

	var draft *llmExtractedDraft
	engineUsed := "hybrid-nlp-fallback"

	if strings.TrimSpace(s.cfg.GeminiAPIKey) != "" {
		llmCtx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
		defer cancel()
		if parsed, err := s.callGeminiStructured(llmCtx, cleanText, products); err == nil && parsed != nil {
			draft = parsed
			engineUsed = fmt.Sprintf("gemini-ai (%s)", s.cfg.GeminiModel)
		}
	}

	if draft == nil {
		draft = s.parseDeterministicIndonesianChat(cleanText, products)
	}

	resp := s.mapDraftToInventory(draft, products)
	resp.ParserEngine = engineUsed
	resp.LatencyMs = time.Since(start).Milliseconds()
	if resp.LatencyMs == 0 {
		resp.LatencyMs = 1
	}

	return resp, nil
}

func (s *AIOrderParserService) callGeminiStructured(
	ctx context.Context,
	rawText string,
	catalog []domain.Product,
) (*llmExtractedDraft, error) {
	var catalogLines []string
	for _, p := range catalog {
		catalogLines = append(catalogLines, fmt.Sprintf("- SKU: %s | Name: %s | Price: %.0f", p.SKU, p.Name, p.Price))
	}

	prompt := fmt.Sprintf(
		"Extract structured Indonesian e-commerce order details from this WhatsApp/Live chat.\nCatalog:\n%s\n\nChat:\n%s\n\nReturn strict JSON matching schema: {\"customer_name\":string,\"customer_phone\":string,\"shipping_address\":string,\"courier_code\":\"JNE\"|\"JNT\"|\"SICEPAT\",\"items\":[{\"sku\":string,\"product_name\":string,\"qty\":number}]}",
		strings.Join(catalogLines, "\n"),
		rawText,
	)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
			"temperature":      0.1,
		},
	}

	payloadBytes, _ := json.Marshal(reqBody)
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		s.cfg.GeminiModel,
		s.cfg.GeminiAPIKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini status %d", res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, err
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty gemini candidates")
	}

	var draft llmExtractedDraft
	if err := json.Unmarshal([]byte(geminiResp.Candidates[0].Content.Parts[0].Text), &draft); err != nil {
		return nil, err
	}
	return &draft, nil
}

var (
	phoneRegex   = regexp.MustCompile(`(\+62|62|0)8[0-9]{7,12}`)
	courierRegex = regexp.MustCompile(`(?i)\b(jne|j&t|jnt|sicepat|anteraja|pos)\b`)
	nameRegex    = regexp.MustCompile(`(?i)(?:kirim\s+ke|atas\s+nama|a\.?n\.?|penerima\s*:?|nama\s*:?)\s+([A-Za-z][A-Za-z\s\.]{2,40}?)(?:\s+(?:\+62|62|0)8|,|\n|$)`)
	addrRegex    = regexp.MustCompile(`(?i)((?:jl\.?|jalan|gg\.?|gang|perum(?:ahan)?|komp(?:lek)?|apartemen|gedung)\s+[^,\n]+(?:,\s*[^,\n]+)*)`)
	qtyAfterRe   = regexp.MustCompile(`(?i)(\d+)\s*(?:pcs|pc|buah|biji|potong|pasang|unit|x)\b`)
	qtyBeforeRe  = regexp.MustCompile(`(?i)\b(\d+)\s*x\s*`)
)

func (s *AIOrderParserService) parseDeterministicIndonesianChat(rawText string, catalog []domain.Product) *llmExtractedDraft {
	draft := &llmExtractedDraft{
		CustomerName:    "Pelanggan WhatsApp",
		CustomerPhone:   "081200000000",
		ShippingAddress: "Alamat belum lengkap (mohon konfirmasi)",
		CourierCode:     "JNE",
	}

	// 1. Extract Phone Number
	if m := phoneRegex.FindString(rawText); m != "" {
		if strings.HasPrefix(m, "+62") {
			m = "0" + strings.TrimPrefix(m, "+62")
		} else if strings.HasPrefix(m, "62") {
			m = "0" + strings.TrimPrefix(m, "62")
		}
		draft.CustomerPhone = m
	}

	// 2. Extract Courier Code
	if cm := courierRegex.FindString(rawText); cm != "" {
		upper := strings.ToUpper(cm)
		if upper == "J&T" {
			upper = "JNT"
		}
		draft.CourierCode = upper
	}

	// 3. Extract Customer Name
	if nm := nameRegex.FindStringSubmatch(rawText); len(nm) > 1 {
		draft.CustomerName = strings.TrimSpace(strings.Trim(nm[1], ",- "))
	}

	// 4. Extract Shipping Address
	if am := addrRegex.FindStringSubmatch(rawText); len(am) > 1 {
		addr := am[1]
		// Strip trailing courier clause like "via JNE REG ya"
		viaIdx := regexp.MustCompile(`(?i)\s+(?:via|pakai|pake|kurir|ekspedisi)\s+`).FindStringIndex(addr)
		if viaIdx != nil {
			addr = addr[:viaIdx[0]]
		}
		draft.ShippingAddress = strings.TrimSpace(strings.Trim(addr, ",. "))
	}

	// 5. Match Products & Quantities against Merchant Catalog
	// Split order clauses by conjunctions ("sama", "dan", "+", ",", "&") before "kirim ke"
	orderPortion := rawText
	if kirimIdx := regexp.MustCompile(`(?i)\b(?:kirim\s+ke|alamat|penerima)\b`).FindStringIndex(rawText); kirimIdx != nil {
		orderPortion = rawText[:kirimIdx[0]]
	}

	clauses := regexp.MustCompile(`(?i)(?:\bsama\b|\bdan\b|\bplus\b|,|\+|&|\n)`).Split(orderPortion, -1)
	seenSKUs := make(map[string]int)

	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}

		bestProd, score := matchBestProduct(clause, catalog)
		if bestProd == nil || score < 2 {
			continue
		}

		qty := extractQuantity(clause)
		if idx, exists := seenSKUs[bestProd.SKU]; exists {
			draft.Items[idx].Qty += qty
		} else {
			seenSKUs[bestProd.SKU] = len(draft.Items)
			draft.Items = append(draft.Items, struct {
				SKU         string `json:"sku"`
				ProductName string `json:"product_name"`
				Qty         int    `json:"qty"`
			}{
				SKU:         bestProd.SKU,
				ProductName: bestProd.Name,
				Qty:         qty,
			})
		}
	}

	// Fallback whole-text scan if clause splitting missed a product
	if len(draft.Items) == 0 {
		bestProd, score := matchBestProduct(orderPortion, catalog)
		if bestProd != nil && score >= 1 {
			draft.Items = append(draft.Items, struct {
				SKU         string `json:"sku"`
				ProductName string `json:"product_name"`
				Qty         int    `json:"qty"`
			}{
				SKU:         bestProd.SKU,
				ProductName: bestProd.Name,
				Qty:         extractQuantity(orderPortion),
			})
		}
	}

	return draft
}

func matchBestProduct(text string, catalog []domain.Product) (*domain.Product, int) {
	lowerText := strings.ToLower(text)
	var best *domain.Product
	bestScore := 0

	for i := range catalog {
		p := &catalog[i]
		if strings.Contains(lowerText, strings.ToLower(p.SKU)) {
			return p, 100
		}

		tokens := strings.Fields(strings.ToLower(p.Name))
		score := 0
		for _, tok := range tokens {
			tok = strings.Trim(tok, "(),-")
			if len(tok) == 0 {
				continue
			}
			// Exact word or strong substring match
			if strings.Contains(lowerText, tok) {
				if len(tok) >= 4 {
					score += 2
				} else {
					score += 1
				}
			}
		}

		if score > bestScore {
			bestScore = score
			best = p
		}
	}
	return best, bestScore
}

func extractQuantity(clause string) int {
	if m := qtyAfterRe.FindStringSubmatch(clause); len(m) > 1 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n
		}
	}
	if m := qtyBeforeRe.FindStringSubmatch(clause); len(m) > 1 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

func (s *AIOrderParserService) mapDraftToInventory(
	draft *llmExtractedDraft,
	catalog []domain.Product,
) *domain.AIParseResponseDTO {
	bySKU := make(map[string]domain.Product, len(catalog))
	for _, p := range catalog {
		bySKU[strings.ToUpper(p.SKU)] = p
	}

	courier := strings.ToUpper(strings.TrimSpace(draft.CourierCode))
	if courier == "" {
		courier = "JNE"
	}
	if courier == "J&T" {
		courier = "JNT"
	}

	resp := &domain.AIParseResponseDTO{
		CustomerName:    draft.CustomerName,
		CustomerPhone:   draft.CustomerPhone,
		ShippingAddress: draft.ShippingAddress,
		CourierCode:     courier,
		Items:           make([]domain.AIParsedItemDTO, 0, len(draft.Items)),
	}

	for _, item := range draft.Items {
		qty := item.Qty
		if qty <= 0 {
			qty = 1
		}

		var matched *domain.Product
		if p, ok := bySKU[strings.ToUpper(item.SKU)]; ok {
			matched = &p
		} else if best, _ := matchBestProduct(item.ProductName+" "+item.SKU, catalog); best != nil {
			matched = best
		}

		if matched == nil {
			continue
		}

		subtotal := matched.Price * float64(qty)
		sufficient := matched.AvailableStock >= qty
		warning := ""
		if !sufficient {
			resp.HasInsufficientStock = true
			warning = fmt.Sprintf("Stok kurang! Tersedia %d pcs, diminta %d pcs", matched.AvailableStock, qty)
		}

		resp.EstimatedTotal += subtotal
		resp.Items = append(resp.Items, domain.AIParsedItemDTO{
			ProductID:          matched.ID,
			SKU:                matched.SKU,
			MatchedProductName: matched.Name,
			Qty:                qty,
			UnitPrice:          matched.Price,
			Subtotal:           subtotal,
			AvailableStock:     matched.AvailableStock,
			StockSufficient:    sufficient,
			StockWarning:       warning,
		})
	}

	return resp
}

