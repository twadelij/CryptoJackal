package backtest

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
)

// ExportCSV writes backtest trades as CSV to the given writer
func ExportCSV(r Result, w io.Writer) error {
	fw := csv.NewWriter(w)
	defer fw.Flush()

	header := []string{
		"entry_index", "exit_index", "entry_price", "exit_price",
		"amount", "entry_fee", "exit_fee", "profit_loss",
		"strategy", "entry_time", "exit_time", "exit_reason",
	}
	if err := fw.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, t := range r.Trades {
		row := []string{
			strconv.Itoa(t.EntryIndex),
			strconv.Itoa(t.ExitIndex),
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.8f", t.ExitPrice),
			fmt.Sprintf("%.8f", t.Amount),
			fmt.Sprintf("%.4f", t.EntryFee),
			fmt.Sprintf("%.4f", t.ExitFee),
			fmt.Sprintf("%.4f", t.ProfitLoss),
			t.Strategy,
			t.EntryTime.Format(time.RFC3339),
			t.ExitTime.Format(time.RFC3339),
			t.ExitReason,
		}
		if err := fw.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// ExportJSON writes the full backtest result as JSON to the given writer
func ExportJSON(r Result, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// ExportSummaryCSV writes a summary of the backtest as a single CSV row
func ExportSummaryCSV(r Result, w io.Writer) error {
	fw := csv.NewWriter(w)
	defer fw.Flush()

	header := []string{
		"total_trades", "winning_trades", "losing_trades",
		"win_rate", "total_pnl", "max_drawdown",
		"sharpe_ratio", "final_balance", "total_fees",
	}
	if err := fw.Write(header); err != nil {
		return err
	}

	row := []string{
		strconv.Itoa(r.TotalTrades),
		strconv.Itoa(r.WinningTrades),
		strconv.Itoa(r.LosingTrades),
		fmt.Sprintf("%.2f", r.WinRate),
		fmt.Sprintf("%.4f", r.TotalProfitLoss),
		fmt.Sprintf("%.2f", r.MaxDrawdown),
		fmt.Sprintf("%.2f", r.SharpeRatio),
		fmt.Sprintf("%.4f", r.FinalBalance),
		fmt.Sprintf("%.4f", r.TotalFees),
	}
	return fw.Write(row)
}
