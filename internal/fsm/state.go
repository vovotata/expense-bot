package fsm

import (
	"time"

	"expense-bot/internal/domain"
)

type Step int

const (
	StepIdle Step = iota
	StepExpenseType
	StepPaymentMethod  // FLOW A only
	StepAddress        // FLOW A only
	StepAmount         // FLOW A only
	StepAntiqueAccount // FLOW B only
	StepComment
	StepConfirm
)

func (s Step) String() string {
	switch s {
	case StepIdle:
		return "idle"
	case StepExpenseType:
		return "expense_type"
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
	default:
		return "unknown"
	}
}

type WizardState struct {
	UserID        int64
	CurrentStep   Step
	FlowType      string // "A" or "B"
	ExpenseType   domain.ExpenseType
	PaymentMethod domain.PaymentMethod
	Address       string
	AddressPhoto  string // Telegram file_id
	Amount        string
	AntiqueAcct   string
	Comment       string
	StartedAt     time.Time
	LastActiveAt  time.Time
}

// NextStep returns the next step based on current flow.
func (ws *WizardState) NextStep() Step {
	switch ws.CurrentStep {
	case StepExpenseType:
		if ws.FlowType == "B" {
			return StepAntiqueAccount
		}
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
	default:
		return StepIdle
	}
}

// PrevStep returns the previous step for edit navigation.
func (ws *WizardState) PrevStep() Step {
	switch ws.CurrentStep {
	case StepPaymentMethod:
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
	default:
		return StepIdle
	}
}
