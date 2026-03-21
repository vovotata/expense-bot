package domain

type ExpenseType string

const (
	ExpenseAgentki        ExpenseType = "agentki"
	ExpenseAdpos          ExpenseType = "adpos"
	ExpenseAntiqueService ExpenseType = "antique_service"
	ExpenseOtherService   ExpenseType = "other_service"
	ExpenseSetups         ExpenseType = "setups"
	ExpenseProxy          ExpenseType = "proxy"
)

func (e ExpenseType) Label() string {
	switch e {
	case ExpenseAgentki:
		return "Агентки"
	case ExpenseAdpos:
		return "Адпос"
	case ExpenseAntiqueService:
		return "Сервис в Антике"
	case ExpenseOtherService:
		return "Другие сервисы"
	case ExpenseSetups:
		return "Сетапы"
	case ExpenseProxy:
		return "Прокси"
	default:
		return string(e)
	}
}

type PaymentMethod string

const (
	PaymentUSDT PaymentMethod = "usdt"
	PaymentTRX  PaymentMethod = "trx"
	PaymentCard PaymentMethod = "card"
	PaymentNone PaymentMethod = "none"
)

func (p PaymentMethod) Label() string {
	switch p {
	case PaymentUSDT:
		return "USDT"
	case PaymentTRX:
		return "TRX"
	case PaymentCard:
		return "Карта"
	case PaymentNone:
		return "—"
	default:
		return string(p)
	}
}

type RequestStatus string

const (
	StatusPending   RequestStatus = "pending"
	StatusApproved  RequestStatus = "approved"
	StatusPaid      RequestStatus = "paid"
	StatusRejected  RequestStatus = "rejected"
	StatusCancelled RequestStatus = "cancelled"
)

func (s RequestStatus) Label() string {
	switch s {
	case StatusPending:
		return "Ожидает"
	case StatusApproved:
		return "Одобрена"
	case StatusPaid:
		return "Оплачена"
	case StatusRejected:
		return "Отклонена"
	case StatusCancelled:
		return "Отменена"
	default:
		return string(s)
	}
}
