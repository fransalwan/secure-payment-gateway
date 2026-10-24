package reconciliation

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
)

var (
	ErrEmptySettlementFile = errors.New("settlement file is empty")
	ErrInvalidCSVFormat    = errors.New("invalid settlement CSV format")
)

// ParseBankSettlementCSV parses bank settlement reports in CSV format.
// Expected CSV columns: reference_id, amount, bank_status, timestamp
func ParseBankSettlementCSV(r io.Reader) ([]*domain.BankSettlementRecord, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrEmptySettlementFile
		}
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate expected columns
	if len(header) < 3 {
		return nil, ErrInvalidCSVFormat
	}

	var records []*domain.BankSettlementRecord

	lineNum := 1
	for {
		lineNum++
		row, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("error reading CSV line %d: %w", lineNum, err)
		}

		if len(row) < 3 {
			continue
		}

		refID := strings.TrimSpace(row[0])
		if refID == "" {
			continue
		}

		amountStr := strings.TrimSpace(row[1])
		amount, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount '%s' at line %d: %w", amountStr, lineNum, err)
		}

		bankStatus := strings.ToUpper(strings.TrimSpace(row[2]))

		var t time.Time
		if len(row) >= 4 && strings.TrimSpace(row[3]) != "" {
			t, _ = time.Parse(time.RFC3339, strings.TrimSpace(row[3]))
		}
		if t.IsZero() {
			t = time.Now().UTC()
		}

		records = append(records, &domain.BankSettlementRecord{
			ReferenceID: refID,
			Amount:      amount,
			BankStatus:  bankStatus,
			Timestamp:   t,
		})
	}

	return records, nil
}

