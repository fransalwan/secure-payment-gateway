package reconciliation_test

import (
	"strings"
	"testing"

	"github.com/fransalwan/secure-payment-gateway/pkg/reconciliation"
)

func TestParseBankSettlementCSV_Success(t *testing.T) {
	csvData := `reference_id,amount,bank_status,timestamp
REF-001,50000,SUCCESS,2026-10-25T01:00:00Z
REF-002,125000,SUCCESS,2026-10-25T01:05:00Z
REF-003,75000,SUCCESS,2026-10-25T01:10:00Z`

	records, err := reconciliation.ParseBankSettlementCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error parsing CSV: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	if records[0].ReferenceID != "REF-001" || records[0].Amount != 50000 {
		t.Errorf("record 0 mismatch: %+v", records[0])
	}

	if records[1].ReferenceID != "REF-002" || records[1].Amount != 125000 {
		t.Errorf("record 1 mismatch: %+v", records[1])
	}
}

func TestParseBankSettlementCSV_InvalidAmount(t *testing.T) {
	csvData := `reference_id,amount,bank_status,timestamp
REF-001,INVALID_AMOUNT,SUCCESS,2026-10-25T01:00:00Z`

	_, err := reconciliation.ParseBankSettlementCSV(strings.NewReader(csvData))
	if err == nil {
		t.Fatal("expected error for invalid amount, got nil")
	}
}

func TestParseBankSettlementCSV_Empty(t *testing.T) {
	_, err := reconciliation.ParseBankSettlementCSV(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty CSV, got nil")
	}
}
