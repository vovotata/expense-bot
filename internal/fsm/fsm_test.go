package fsm

import (
	"context"
	"testing"
	"time"

	"expense-bot/internal/domain"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore(5 * time.Minute)
	defer store.Stop()
	ctx := context.Background()

	state := &WizardState{
		UserID:      123,
		CurrentStep: StepExpenseType,
		FlowType:    "A",
	}

	if err := store.Set(ctx, state); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := store.Get(ctx, 123)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected state, got nil")
	}
	if got.CurrentStep != StepExpenseType {
		t.Errorf("expected StepExpenseType, got %v", got.CurrentStep)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore(5 * time.Minute)
	defer store.Stop()
	ctx := context.Background()

	state := &WizardState{UserID: 456, CurrentStep: StepComment}
	store.Set(ctx, state)
	store.Delete(ctx, 456)

	got, _ := store.Get(ctx, 456)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemoryStore_TTLExpiry(t *testing.T) {
	store := NewMemoryStore(1 * time.Millisecond)
	defer store.Stop()
	ctx := context.Background()

	state := &WizardState{UserID: 789, CurrentStep: StepAmount}
	store.Set(ctx, state)

	time.Sleep(5 * time.Millisecond)

	got, _ := store.Get(ctx, 789)
	if got != nil {
		t.Error("expected nil after TTL expiry")
	}
}

func TestMemoryStore_GetNonExistent(t *testing.T) {
	store := NewMemoryStore(5 * time.Minute)
	defer store.Stop()
	ctx := context.Background()

	got, err := store.Get(ctx, 999)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestWizardState_NextStep_FlowA(t *testing.T) {
	tests := []struct {
		current  Step
		expected Step
	}{
		{StepExpenseType, StepPaymentMethod},
		{StepPaymentMethod, StepAddress},
		{StepAddress, StepAmount},
		{StepAmount, StepComment},
		{StepComment, StepConfirm},
	}

	for _, tt := range tests {
		ws := &WizardState{FlowType: "A", CurrentStep: tt.current}
		got := ws.NextStep()
		if got != tt.expected {
			t.Errorf("FlowA: NextStep(%s) = %s, want %s", tt.current, got, tt.expected)
		}
	}
}

func TestWizardState_NextStep_FlowB(t *testing.T) {
	tests := []struct {
		current  Step
		expected Step
	}{
		{StepExpenseType, StepAntiqueAccount},
		{StepAntiqueAccount, StepComment},
		{StepComment, StepConfirm},
	}

	for _, tt := range tests {
		ws := &WizardState{FlowType: "B", CurrentStep: tt.current}
		got := ws.NextStep()
		if got != tt.expected {
			t.Errorf("FlowB: NextStep(%s) = %s, want %s", tt.current, got, tt.expected)
		}
	}
}

func TestWizardState_PrevStep_FlowA(t *testing.T) {
	tests := []struct {
		current  Step
		expected Step
	}{
		{StepConfirm, StepComment},
		{StepComment, StepAmount},
		{StepAmount, StepAddress},
		{StepAddress, StepPaymentMethod},
		{StepPaymentMethod, StepExpenseType},
	}

	for _, tt := range tests {
		ws := &WizardState{FlowType: "A", CurrentStep: tt.current}
		got := ws.PrevStep()
		if got != tt.expected {
			t.Errorf("FlowA: PrevStep(%s) = %s, want %s", tt.current, got, tt.expected)
		}
	}
}

func TestWizardState_PrevStep_FlowB(t *testing.T) {
	tests := []struct {
		current  Step
		expected Step
	}{
		{StepConfirm, StepComment},
		{StepComment, StepAntiqueAccount},
		{StepAntiqueAccount, StepExpenseType},
	}

	for _, tt := range tests {
		ws := &WizardState{FlowType: "B", CurrentStep: tt.current}
		got := ws.PrevStep()
		if got != tt.expected {
			t.Errorf("FlowB: PrevStep(%s) = %s, want %s", tt.current, got, tt.expected)
		}
	}
}

func TestStepString(t *testing.T) {
	tests := []struct {
		step Step
		want string
	}{
		{StepIdle, "idle"},
		{StepExpenseType, "expense_type"},
		{StepPaymentMethod, "payment_method"},
		{StepAddress, "address"},
		{StepAmount, "amount"},
		{StepAntiqueAccount, "antique_account"},
		{StepComment, "comment"},
		{StepConfirm, "confirm"},
	}
	for _, tt := range tests {
		if got := tt.step.String(); got != tt.want {
			t.Errorf("Step(%d).String() = %q, want %q", tt.step, got, tt.want)
		}
	}
}

func TestFlowTypeAssignment(t *testing.T) {
	ws := &WizardState{}

	// Antique service -> Flow B
	ws.ExpenseType = domain.ExpenseAntiqueService
	ws.FlowType = "B"
	ws.CurrentStep = StepExpenseType

	next := ws.NextStep()
	if next != StepAntiqueAccount {
		t.Errorf("expected StepAntiqueAccount for Flow B, got %s", next)
	}

	// Agentki -> StepAgentName
	ws.ExpenseType = domain.ExpenseAgentki
	ws.FlowType = "A"

	next = ws.NextStep()
	if next != StepAgentName {
		t.Errorf("expected StepAgentName for Agentki, got %s", next)
	}

	// Other types -> StepPaymentMethod
	ws.ExpenseType = domain.ExpenseAdpos
	next = ws.NextStep()
	if next != StepPaymentMethod {
		t.Errorf("expected StepPaymentMethod for non-Agentki Flow A, got %s", next)
	}
}
