package postgres

import (
	"context"
	"fmt"
	"time"

	"expense-bot/internal/config"
	"expense-bot/internal/domain"
	db "expense-bot/internal/storage/postgres/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func New(ctx context.Context, cfg *config.Config) (*Store, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("postgres.New: parse config: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.DBMaxOpenConns)
	poolCfg.MinConns = int32(cfg.DBMaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.DBConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres.New: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres.New: ping: %w", err)
	}

	return &Store{
		pool:    pool,
		queries: db.New(pool),
	}, nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) Close() {
	s.pool.Close()
}

// --- Users ---

func (s *Store) UpsertUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	row, err := s.queries.UpsertUser(ctx, db.UpsertUserParams{
		ID:        user.ID,
		Username:  pgtext(user.Username),
		FirstName: user.FirstName,
		LastName:  pgtext(user.LastName),
	})
	if err != nil {
		return nil, fmt.Errorf("storage.UpsertUser: %w", err)
	}
	return userFromDB(row), nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage.GetUserByID: %w", err)
	}
	return userFromDB(row), nil
}

func (s *Store) IsUserBlocked(ctx context.Context, id int64) (bool, error) {
	blocked, err := s.queries.IsUserBlocked(ctx, id)
	if err != nil {
		return false, fmt.Errorf("storage.IsUserBlocked: %w", err)
	}
	return blocked, nil
}

// --- Requests ---

func (s *Store) CreateRequest(ctx context.Context, req *domain.Request) (*domain.Request, error) {
	var amount pgtype.Numeric
	if !req.Amount.IsZero() {
		amount = decimalToNumeric(req.Amount)
	}

	row, err := s.queries.CreateRequest(ctx, db.CreateRequestParams{
		UserID:         req.UserID,
		ExpenseType:    db.ExpenseType(req.ExpenseType),
		PaymentMethod:  db.NullPaymentMethod{PaymentMethod: db.PaymentMethod(req.PaymentMethod), Valid: req.PaymentMethod != ""},
		Address:        pgtext(req.Address),
		AddressPhoto:   pgtext(req.AddressPhoto),
		Amount:         amount,
		AntiqueAccount: pgtext(req.AntiqueAccount),
		Comment:        req.Comment,
		FlowType:      req.FlowType,
		TgMessageID:    pgint8(req.TgMessageID),
	})
	if err != nil {
		return nil, fmt.Errorf("storage.CreateRequest: %w", err)
	}
	return requestFromDB(row), nil
}

func (s *Store) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.Request, error) {
	row, err := s.queries.GetRequestByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage.GetRequestByID: %w", err)
	}
	return requestFromDB(row), nil
}

func (s *Store) ListRequestsByUser(ctx context.Context, userID int64, limit, offset int32) ([]*domain.Request, error) {
	rows, err := s.queries.ListRequestsByUser(ctx, db.ListRequestsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.ListRequestsByUser: %w", err)
	}
	result := make([]*domain.Request, len(rows))
	for i, row := range rows {
		result[i] = requestFromDB(row)
	}
	return result, nil
}

func (s *Store) ListPendingRequests(ctx context.Context) ([]*domain.Request, error) {
	rows, err := s.queries.ListPendingRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.ListPendingRequests: %w", err)
	}
	result := make([]*domain.Request, len(rows))
	for i, row := range rows {
		result[i] = &domain.Request{
			ID:             row.ID,
			UserID:         row.UserID,
			ExpenseType:    domain.ExpenseType(row.ExpenseType),
			PaymentMethod:  domain.PaymentMethod(nullPaymentStr(row.PaymentMethod)),
			Address:        textStr(row.Address),
			AddressPhoto:   textStr(row.AddressPhoto),
			Amount:         numericToDecimal(row.Amount),
			AntiqueAccount: textStr(row.AntiqueAccount),
			Comment:        row.Comment,
			Status:         domain.RequestStatus(row.Status),
			FlowType:       row.FlowType,
			TgMessageID:    int8Val(row.TgMessageID),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		}
	}
	return result, nil
}

func (s *Store) UpdateRequestStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus) (*domain.Request, error) {
	row, err := s.queries.UpdateRequestStatus(ctx, db.UpdateRequestStatusParams{
		ID:     id,
		Status: db.RequestStatus(status),
	})
	if err != nil {
		return nil, fmt.Errorf("storage.UpdateRequestStatus: %w", err)
	}
	return requestFromDB(row), nil
}

func (s *Store) CountRequestsByStatus(ctx context.Context) (map[domain.RequestStatus]int64, error) {
	rows, err := s.queries.CountRequestsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.CountRequestsByStatus: %w", err)
	}
	result := make(map[domain.RequestStatus]int64)
	for _, row := range rows {
		result[domain.RequestStatus(row.Status)] = row.Count
	}
	return result, nil
}

// --- Email Accounts ---

func (s *Store) CreateEmailAccount(ctx context.Context, acc *domain.EmailAccount) (*domain.EmailAccount, error) {
	row, err := s.queries.CreateEmailAccount(ctx, db.CreateEmailAccountParams{
		UserID:      acc.UserID,
		Email:       acc.Email,
		ImapServer:  acc.IMAPServer,
		PasswordEnc: acc.PasswordEnc,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.CreateEmailAccount: %w", err)
	}
	return &domain.EmailAccount{
		ID:         row.ID,
		UserID:     row.UserID,
		Email:      row.Email,
		IMAPServer: row.ImapServer,
		IsActive:   row.IsActive,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

func (s *Store) ListEmailAccountsByUser(ctx context.Context, userID int64) ([]*domain.EmailAccount, error) {
	rows, err := s.queries.ListEmailAccountsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("storage.ListEmailAccountsByUser: %w", err)
	}
	result := make([]*domain.EmailAccount, len(rows))
	for i, row := range rows {
		var lastConn *time.Time
		if row.LastConnected.Valid {
			t := row.LastConnected.Time
			lastConn = &t
		}
		var lastErr *string
		if row.LastError.Valid {
			lastErr = &row.LastError.String
		}
		result[i] = &domain.EmailAccount{
			ID:            row.ID,
			UserID:        row.UserID,
			Email:         row.Email,
			IMAPServer:    row.ImapServer,
			IsActive:      row.IsActive,
			LastConnected: lastConn,
			LastError:     lastErr,
			CreatedAt:     row.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *Store) GetActiveEmailAccounts(ctx context.Context) ([]*domain.EmailAccount, error) {
	rows, err := s.queries.GetActiveEmailAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.GetActiveEmailAccounts: %w", err)
	}
	result := make([]*domain.EmailAccount, len(rows))
	for i, row := range rows {
		result[i] = &domain.EmailAccount{
			ID:          row.ID,
			UserID:      row.UserID,
			Email:       row.Email,
			IMAPServer:  row.ImapServer,
			PasswordEnc: row.PasswordEnc,
			IsActive:    row.IsActive,
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *Store) UpdateEmailAccountStatus(ctx context.Context, id int64, lastConnected *string, lastError *string) error {
	var lc pgtype.Timestamptz
	if lastConnected != nil {
		t, err := time.Parse(time.RFC3339, *lastConnected)
		if err == nil {
			lc = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	var le pgtype.Text
	if lastError != nil {
		le = pgtype.Text{String: *lastError, Valid: true}
	}
	return s.queries.UpdateEmailAccountStatus(ctx, db.UpdateEmailAccountStatusParams{
		ID:            id,
		LastConnected: lc,
		LastError:     le,
	})
}

func (s *Store) DeactivateEmailAccount(ctx context.Context, id int64, userID int64) error {
	return s.queries.DeactivateEmailAccount(ctx, db.DeactivateEmailAccountParams{ID: id, UserID: userID})
}

func (s *Store) DeleteEmailAccount(ctx context.Context, id int64, userID int64) error {
	return s.queries.DeleteEmailAccount(ctx, db.DeleteEmailAccountParams{ID: id, UserID: userID})
}

func (s *Store) GetEmailAccountPassword(ctx context.Context, id int64) ([]byte, error) {
	return s.queries.GetEmailAccountPassword(ctx, id)
}

func (s *Store) CountEmailAccountsByUser(ctx context.Context, userID int64) (int64, error) {
	return s.queries.CountEmailAccountsByUser(ctx, userID)
}

// --- Email Codes ---

func (s *Store) CreateEmailCode(ctx context.Context, code *domain.EmailCode) (*domain.EmailCode, error) {
	row, err := s.queries.CreateEmailCode(ctx, db.CreateEmailCodeParams{
		EmailAccountID: code.EmailAccountID,
		UserID:         code.UserID,
		Sender:         code.Sender,
		Subject:        pgtext(code.Subject),
		Code:           code.Code,
		RuleName:       pgtext(code.RuleName),
		RawBodyHash:    pgtext(code.RawBodyHash),
		TgMessageID:    pgint8(code.TgMessageID),
		ReceivedAt:     pgtype.Timestamptz{Time: code.ReceivedAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("storage.CreateEmailCode: %w", err)
	}
	return &domain.EmailCode{
		ID:             row.ID,
		EmailAccountID: row.EmailAccountID,
		UserID:         row.UserID,
		Sender:         row.Sender,
		Code:           row.Code,
		ReceivedAt:     row.ReceivedAt.Time,
		CreatedAt:      row.CreatedAt.Time,
	}, nil
}

func (s *Store) ListRecentCodesByUser(ctx context.Context, userID int64, limit int32) ([]*domain.EmailCode, error) {
	rows, err := s.queries.ListRecentCodesByUser(ctx, db.ListRecentCodesByUserParams{
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.ListRecentCodesByUser: %w", err)
	}
	result := make([]*domain.EmailCode, len(rows))
	for i, row := range rows {
		result[i] = &domain.EmailCode{
			ID:             row.ID,
			EmailAccountID: row.EmailAccountID,
			UserID:         row.UserID,
			Sender:         row.Sender,
			Subject:        textStr(row.Subject),
			Code:           row.Code,
			RuleName:       textStr(row.RuleName),
			ReceivedAt:     row.ReceivedAt.Time,
			CreatedAt:      row.CreatedAt.Time,
			Email:          row.Email,
		}
	}
	return result, nil
}

func (s *Store) CodeExistsByBodyHash(ctx context.Context, hash string) (bool, error) {
	exists, err := s.queries.CodeExistsByBodyHash(ctx, pgtext(hash))
	if err != nil {
		return false, fmt.Errorf("storage.CodeExistsByBodyHash: %w", err)
	}
	return exists, nil
}

func (s *Store) DeleteOldCodes(ctx context.Context) (int64, error) {
	return s.queries.DeleteOldCodes(ctx)
}

// --- Helpers ---

func pgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func pgint8(i int64) pgtype.Int8 {
	if i == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: i, Valid: true}
}

func textStr(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func int8Val(i pgtype.Int8) int64 {
	if i.Valid {
		return i.Int64
	}
	return 0
}

func nullPaymentStr(p db.NullPaymentMethod) string {
	if p.Valid {
		return string(p.PaymentMethod)
	}
	return ""
}

func decimalToNumeric(d decimal.Decimal) pgtype.Numeric {
	// Use the decimal string representation to set numeric
	var n pgtype.Numeric
	_ = n.Scan(d.String())
	return n
}

func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	// Convert via string
	val, _ := n.Value()
	if val == nil {
		return decimal.Zero
	}
	d, _ := decimal.NewFromString(fmt.Sprintf("%v", val))
	return d
}

func userFromDB(row db.User) *domain.User {
	return &domain.User{
		ID:        row.ID,
		Username:  textStr(row.Username),
		FirstName: row.FirstName,
		LastName:  textStr(row.LastName),
		IsBlocked: row.IsBlocked,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func requestFromDB(row db.Request) *domain.Request {
	return &domain.Request{
		ID:             row.ID,
		UserID:         row.UserID,
		ExpenseType:    domain.ExpenseType(row.ExpenseType),
		PaymentMethod:  domain.PaymentMethod(nullPaymentStr(row.PaymentMethod)),
		Address:        textStr(row.Address),
		AddressPhoto:   textStr(row.AddressPhoto),
		Amount:         numericToDecimal(row.Amount),
		AntiqueAccount: textStr(row.AntiqueAccount),
		Comment:        row.Comment,
		Status:         domain.RequestStatus(row.Status),
		FlowType:       row.FlowType,
		TgMessageID:    int8Val(row.TgMessageID),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}
