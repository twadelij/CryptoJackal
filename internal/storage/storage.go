package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/twadelij/cryptojackal/internal/models"
	"modernc.org/sqlite"
)

func init() {
	// Register the driver name
	_ = sqlite.Driver{}
}

// Storage handles all database operations
type Storage struct {
	db     *sql.DB
	dbPath string
}

// New creates a new storage instance with the given database path
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Storage{db: db, dbPath: dbPath}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return s, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// DBPath returns the database file path
func (s *Storage) DBPath() string {
	return s.dbPath
}

// migrate creates the database schema
func (s *Storage) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS trades (
	id TEXT PRIMARY KEY,
	token_address TEXT NOT NULL,
	token_symbol TEXT NOT NULL,
	type TEXT NOT NULL,
	amount_in REAL NOT NULL,
	amount_out REAL,
	price REAL NOT NULL,
	profit_loss REAL,
	status TEXT NOT NULL,
	is_paper_trade BOOLEAN NOT NULL,
	executed_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS portfolio (
	id TEXT PRIMARY KEY,
	balance REAL NOT NULL,
	total_value REAL NOT NULL,
	profit_loss REAL NOT NULL,
	initial_balance REAL NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS token_balances (
	portfolio_id TEXT NOT NULL,
	token_address TEXT NOT NULL,
	token_symbol TEXT NOT NULL,
	token_name TEXT,
	balance REAL NOT NULL,
	avg_price REAL NOT NULL,
	current_price REAL NOT NULL,
	PRIMARY KEY (portfolio_id, token_address)
);

CREATE TABLE IF NOT EXISTS config (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at DATETIME NOT NULL
);
`
	_, err := s.db.Exec(schema)
	return err
}

// SaveTrade persists a trade to the database
func (s *Storage) SaveTrade(trade *models.Trade) error {
	_, err := s.db.Exec(
		`INSERT INTO trades (id, token_address, token_symbol, type, amount_in, amount_out, price, profit_loss, status, is_paper_trade, executed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			amount_out = excluded.amount_out,
			profit_loss = excluded.profit_loss,
			status = excluded.status`,
		trade.ID, trade.TokenAddress, trade.TokenSymbol, string(trade.Type),
		trade.AmountIn, trade.AmountOut, trade.Price, trade.ProfitLoss,
		string(trade.Status), trade.IsPaperTrade, trade.ExecutedAt,
	)
	return err
}

// GetTrades retrieves all trades ordered by execution time (newest first)
func (s *Storage) GetTrades(limit int) ([]models.Trade, error) {
	query := `SELECT id, token_address, token_symbol, type, amount_in, amount_out, price, profit_loss, status, is_paper_trade, executed_at
		FROM trades ORDER BY executed_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTrades(rows)
}

// GetTradesByType retrieves trades filtered by type
func (s *Storage) GetTradesByType(tradeType models.TradeType, limit int) ([]models.Trade, error) {
	query := `SELECT id, token_address, token_symbol, type, amount_in, amount_out, price, profit_loss, status, is_paper_trade, executed_at
		FROM trades WHERE type = ? ORDER BY executed_at DESC LIMIT ?`

	rows, err := s.db.Query(query, string(tradeType), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTrades(rows)
}

func scanTrades(rows *sql.Rows) ([]models.Trade, error) {
	var trades []models.Trade
	for rows.Next() {
		var t models.Trade
		var isPaper bool
		err := rows.Scan(
			&t.ID, &t.TokenAddress, &t.TokenSymbol, &t.Type,
			&t.AmountIn, &t.AmountOut, &t.Price, &t.ProfitLoss,
			&t.Status, &isPaper, &t.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}
		t.IsPaperTrade = isPaper
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

// SavePortfolio persists the portfolio and token balances
func (s *Storage) SavePortfolio(portfolio *models.Portfolio, initialBalance float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO portfolio (id, balance, total_value, profit_loss, initial_balance, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			balance = excluded.balance,
			total_value = excluded.total_value,
			profit_loss = excluded.profit_loss,
			updated_at = excluded.updated_at`,
		portfolio.ID, portfolio.Balance, portfolio.TotalValue,
		portfolio.ProfitLoss, initialBalance, time.Now(),
	)
	if err != nil {
		return err
	}

	// Delete old token balances and insert new ones
	_, err = tx.Exec(`DELETE FROM token_balances WHERE portfolio_id = ?`, portfolio.ID)
	if err != nil {
		return err
	}

	for _, tb := range portfolio.TokenBalances {
		_, err = tx.Exec(
			`INSERT INTO token_balances (portfolio_id, token_address, token_symbol, token_name, balance, avg_price, current_price)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			portfolio.ID, tb.Token.Address, tb.Token.Symbol, tb.Token.Name,
			tb.Balance, tb.AvgPrice, tb.Token.Price,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadPortfolio retrieves the portfolio and token balances
func (s *Storage) LoadPortfolio(portfolioID string) (*models.Portfolio, float64, error) {
	var portfolio models.Portfolio
	var initialBalance float64

	err := s.db.QueryRow(
		`SELECT id, balance, total_value, profit_loss, initial_balance, updated_at
		FROM portfolio WHERE id = ?`,
		portfolioID,
	).Scan(
		&portfolio.ID, &portfolio.Balance, &portfolio.TotalValue,
		&portfolio.ProfitLoss, &initialBalance, &portfolio.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}

	portfolio.Currency = "EUR"
	portfolio.TokenBalances = make(map[string]models.TokenBalance)

	rows, err := s.db.Query(
		`SELECT token_address, token_symbol, token_name, balance, avg_price, current_price
		FROM token_balances WHERE portfolio_id = ?`,
		portfolioID,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var tb models.TokenBalance
		err := rows.Scan(
			&tb.Token.Address, &tb.Token.Symbol, &tb.Token.Name,
			&tb.Balance, &tb.AvgPrice, &tb.Token.Price,
		)
		if err != nil {
			return nil, 0, err
		}
		tb.Value = tb.Balance * tb.Token.Price
		portfolio.TokenBalances[tb.Token.Address] = tb
	}

	return &portfolio, initialBalance, rows.Err()
}

// SetConfig stores a config key-value pair
func (s *Storage) SetConfig(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO config (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now(),
	)
	return err
}

// GetConfig retrieves a config value by key
func (s *Storage) GetConfig(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetAllConfigs retrieves all config values
func (s *Storage) GetAllConfigs() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		configs[key] = value
	}
	return configs, rows.Err()
}

// DeleteConfig removes a config key
func (s *Storage) DeleteConfig(key string) error {
	_, err := s.db.Exec(`DELETE FROM config WHERE key = ?`, key)
	return err
}
