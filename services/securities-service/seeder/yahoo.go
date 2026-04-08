package seeder

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

// OptionRow is one option contract to be inserted.
type OptionRow struct {
	StockListingID    int64
	OptionType        string // "CALL" or "PUT"
	StrikePrice       float64
	ImpliedVolatility float64
	OpenInterest      int64
	SettlementDate    time.Time
}

// yahooOptionChain is the minimal structure we need from Yahoo Finance.
type yahooOptionChain struct {
	OptionChain struct {
		Result []struct {
			Options []struct {
				Calls []yahooContract `json:"calls"`
				Puts  []yahooContract `json:"puts"`
			} `json:"options"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"optionChain"`
}

type yahooContract struct {
	Strike            float64 `json:"strike"`
	ImpliedVolatility float64 `json:"impliedVolatility"`
	OpenInterest      int64   `json:"openInterest"`
	Expiration        int64   `json:"expiration"` // Unix timestamp
}

// FetchOptions attempts to fetch option contracts for a ticker from Yahoo Finance.
// Returns nil, nil when the request fails or no data is available (caller should fall back).
func FetchOptions(ticker string, stockListingID int64) ([]OptionRow, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v6/finance/options/%s", ticker)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo options: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("yahoo options: request failed for %s: %v — falling back to generated options", ticker, err)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("yahoo options: status %d for %s — falling back", resp.StatusCode, ticker)
		return nil, nil
	}

	var chain yahooOptionChain
	if err := json.NewDecoder(resp.Body).Decode(&chain); err != nil {
		log.Printf("yahoo options: decode error for %s: %v — falling back", ticker, err)
		return nil, nil
	}

	if len(chain.OptionChain.Result) == 0 || len(chain.OptionChain.Result[0].Options) == 0 {
		return nil, nil
	}

	var rows []OptionRow
	for _, opt := range chain.OptionChain.Result[0].Options {
		for _, c := range opt.Calls {
			rows = append(rows, OptionRow{
				StockListingID:    stockListingID,
				OptionType:        "CALL",
				StrikePrice:       c.Strike,
				ImpliedVolatility: c.ImpliedVolatility,
				OpenInterest:      c.OpenInterest,
				SettlementDate:    time.Unix(c.Expiration, 0).UTC(),
			})
		}
		for _, p := range opt.Puts {
			rows = append(rows, OptionRow{
				StockListingID:    stockListingID,
				OptionType:        "PUT",
				StrikePrice:       p.Strike,
				ImpliedVolatility: p.ImpliedVolatility,
				OpenInterest:      p.OpenInterest,
				SettlementDate:    time.Unix(p.Expiration, 0).UTC(),
			})
		}
	}
	return rows, nil
}

// GenerateOptions creates synthetic option contracts for a stock when Yahoo Finance is unavailable.
//
// Settlement dates:
//   - 6, 12, 18, 24, 30 days from today
//   - then 30, 60, 90, 120, 150, 180 days from today
//
// Strike prices: round(stockPrice) ± 5 at $1 increments.
// For each (settlement × strike): one CALL and one PUT with impliedVolatility = 1.0.
func GenerateOptions(stockListingID int64, stockPrice float64) []OptionRow {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Build settlement date list.
	var settlements []time.Time
	for i := 1; i <= 5; i++ {
		settlements = append(settlements, today.Add(time.Duration(i*6)*24*time.Hour))
	}
	for i := 1; i <= 6; i++ {
		settlements = append(settlements, today.Add(time.Duration(i*30)*24*time.Hour))
	}

	baseStrike := math.Round(stockPrice)
	var rows []OptionRow
	for _, sd := range settlements {
		for delta := -5; delta <= 5; delta++ {
			strike := baseStrike + float64(delta)
			if strike <= 0 {
				continue
			}
			for _, optType := range []string{"CALL", "PUT"} {
				rows = append(rows, OptionRow{
					StockListingID:    stockListingID,
					OptionType:        optType,
					StrikePrice:       strike,
					ImpliedVolatility: 1.0,
					OpenInterest:      0,
					SettlementDate:    sd,
				})
			}
		}
	}
	return rows
}
