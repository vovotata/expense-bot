package storage

import (
	"context"

	"expense-bot/internal/domain"

	"github.com/google/uuid"
)

type Storage interface {
	// Users
	UpsertUser(ctx context.Context, user *domain.User) (*domain.User, error)
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
	IsUserBlocked(ctx context.Context, id int64) (bool, error)
	ListAllActiveUsers(ctx context.Context) ([]*domain.User, error)

	// Requests
	CreateRequest(ctx context.Context, req *domain.Request) (*domain.Request, error)
	GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.Request, error)
	ListRequestsByUser(ctx context.Context, userID int64, limit, offset int32) ([]*domain.Request, error)
	ListPendingRequests(ctx context.Context) ([]*domain.Request, error)
	UpdateRequestStatus(ctx context.Context, id uuid.UUID, status domain.RequestStatus) (*domain.Request, error)
	CountRequestsByStatus(ctx context.Context) (map[domain.RequestStatus]int64, error)

	// Email accounts
	CreateEmailAccount(ctx context.Context, acc *domain.EmailAccount) (*domain.EmailAccount, error)
	ListEmailAccountsByUser(ctx context.Context, userID int64) ([]*domain.EmailAccount, error)
	GetActiveEmailAccounts(ctx context.Context) ([]*domain.EmailAccount, error)
	UpdateEmailAccountStatus(ctx context.Context, id int64, lastConnected *string, lastError *string) error
	DeactivateEmailAccount(ctx context.Context, id int64, userID int64) error
	DeleteEmailAccount(ctx context.Context, id int64, userID int64) error
	GetEmailAccountPassword(ctx context.Context, id int64) ([]byte, error)
	CountEmailAccountsByUser(ctx context.Context, userID int64) (int64, error)

	// Email codes
	CreateEmailCode(ctx context.Context, code *domain.EmailCode) (*domain.EmailCode, error)
	ListRecentCodesByUser(ctx context.Context, userID int64, limit int32) ([]*domain.EmailCode, error)
	ListRecentCodes(ctx context.Context, limit int32) ([]*domain.EmailCode, error)
	CodeExistsByBodyHash(ctx context.Context, hash string) (bool, error)
	DeleteOldCodes(ctx context.Context) (int64, error)

	// Health
	Ping(ctx context.Context) error
	Close()
}
