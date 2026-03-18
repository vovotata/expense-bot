package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	IsBlocked bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Request struct {
	ID             uuid.UUID
	UserID         int64
	ExpenseType    ExpenseType
	PaymentMethod  PaymentMethod
	Address        string
	AddressPhoto   string
	Amount         decimal.Decimal
	AntiqueAccount string
	Comment        string
	Status         RequestStatus
	FlowType       string
	TgMessageID    int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type EmailAccount struct {
	ID            int64
	UserID        int64
	Email         string
	IMAPServer    string
	PasswordEnc   []byte
	IsActive      bool
	LastConnected *time.Time
	LastError     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type EmailCode struct {
	ID             int64
	EmailAccountID int64
	UserID         int64
	Sender         string
	Subject        string
	Code           string
	RuleName       string
	RawBodyHash    string
	TgMessageID    int64
	ReceivedAt     time.Time
	CreatedAt      time.Time
	Email          string // joined from email_accounts
}
