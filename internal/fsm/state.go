package fsm

import (
	"time"

	"expense-bot/internal/domain"
)

type Step int

const (
	StepIdle Step = iota
	StepExpenseType
	StepAgentName      // Агентки only — CROSSGIF/ULD/PREMIUM
	StepProxyProvider  // Прокси only — Proxy6/ProxySeller
	StepPaymentMethod  // FLOW A only
	StepAddress        // FLOW A only
	StepAmount         // FLOW A only
	StepAntiqueAccount // FLOW B only
	StepComment
	StepConfirm

	// Email wizard steps
	StepEmailProvider
	StepEmailAddress
	StepEmailPassword
)

func (s Step) String() string {
	switch s {
	case StepIdle:
		return "idle"
	case StepExpenseType:
		return "expense_type"
	case StepAgentName:
		return "agent_name"
	case StepProxyProvider:
		return "proxy_provider"
	case StepPaymentMethod:
		return "payment_method"
	case StepAddress:
		return "address"
	case StepAmount:
		return "amount"
	case StepAntiqueAccount:
		return "antique_account"
	case StepComment:
		return "comment"
	case StepConfirm:
		return "confirm"
	case StepEmailProvider:
		return "email_provider"
	case StepEmailAddress:
		return "email_address"
	case StepEmailPassword:
		return "email_password"
	default:
		return "unknown"
	}
}

type WizardState struct {
	UserID        int64
	CurrentStep   Step
	FlowType      string // "A", "B", or "email"
	ExpenseType   domain.ExpenseType
	PaymentMethod domain.PaymentMethod
	Address       string
	AddressPhoto  string // Telegram file_id
	Amount        string
	AgentName     string // CROSSGIF/ULD/PREMIUM (for Агентки)
	ProxyProvider string // Proxy6/ProxySeller (for Прокси)
	AntiqueAcct   string
	Comment       string
	StartedAt     time.Time
	LastActiveAt  time.Time

	// Email wizard fields
	EmailProvider string // "gmail", "yandex", "mailru", "outlook", "other"
	EmailIMAPHost string // auto-filled or manual
	EmailAddress  string
	EmailPassword string
}

// NextStep returns the next step based on current flow.
func (ws *WizardState) NextStep() Step {
	switch ws.CurrentStep {
	case StepExpenseType:
		if ws.FlowType == "B" {
			return StepAntiqueAccount
		}
		if ws.ExpenseType == domain.ExpenseAgentki {
			return StepAgentName
		}
		if ws.ExpenseType == domain.ExpenseProxy {
			return StepProxyProvider
		}
		return StepPaymentMethod
	case StepAgentName:
		return StepPaymentMethod
	case StepProxyProvider:
		return StepPaymentMethod
	case StepPaymentMethod:
		return StepAddress
	case StepAddress:
		return StepAmount
	case StepAmount:
		return StepComment
	case StepAntiqueAccount:
		return StepComment
	case StepComment:
		return StepConfirm
	// Email flow
	case StepEmailAddress:
		return StepEmailPassword
	default:
		return StepIdle
	}
}

// PrevStep returns the previous step for edit navigation.
func (ws *WizardState) PrevStep() Step {
	switch ws.CurrentStep {
	case StepAgentName:
		return StepExpenseType
	case StepProxyProvider:
		return StepExpenseType
	case StepPaymentMethod:
		if ws.ExpenseType == domain.ExpenseAgentki {
			return StepAgentName
		}
		if ws.ExpenseType == domain.ExpenseProxy {
			return StepProxyProvider
		}
		return StepExpenseType
	case StepAddress:
		return StepPaymentMethod
	case StepAmount:
		return StepAddress
	case StepAntiqueAccount:
		return StepExpenseType
	case StepComment:
		if ws.FlowType == "B" {
			return StepAntiqueAccount
		}
		return StepAmount
	case StepConfirm:
		return StepComment
	// Email flow — back from email address cancels (it's step 1)
	case StepEmailPassword:
		return StepEmailAddress
	default:
		return StepIdle
	}
}
