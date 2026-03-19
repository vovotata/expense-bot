package emailwatch

import "regexp"

// ParseRule defines how to extract a verification code from an email.
type ParseRule struct {
	Name          string
	SenderFilter  *regexp.Regexp // nil = match any sender
	SubjectFilter *regexp.Regexp // nil = match any subject
	CodePattern   *regexp.Regexp
	CodeGroup     int // capture group index
}

// AutomatedSenderFilter matches typical automated/verification email senders.
// Only emails from these senders will be parsed for codes.
var AutomatedSenderFilter = regexp.MustCompile(
	`(?i)(noreply|no-reply|no\.reply|` +
		`security|verify|verification|` +
		`auth|account|alert|notification|` +
		`info@|support@|admin@|service@|` +
		`do-not-reply|donotreply|` +
		`google\.com|facebook|meta\.com|` +
		`microsoft\.com|apple\.com|` +
		`amazon\.com|twitter\.com|x\.com|` +
		`telegram\.org|instagram\.com|` +
		`binance|bybit|okx|coinbase|kraken)`,
)

// IsAutomatedSender checks if the sender looks like an automated service.
func IsAutomatedSender(from string) bool {
	return AutomatedSenderFilter.MatchString(from)
}

// DefaultRules are checked in order — specific rules first, then generic ones.
var DefaultRules = []ParseRule{
	{
		Name:         "Google",
		SenderFilter: regexp.MustCompile(`(?i)google\.com`),
		CodePattern:  regexp.MustCompile(`(\d{6})`),
		CodeGroup:    1,
	},
	{
		Name:         "Meta/Facebook",
		SenderFilter: regexp.MustCompile(`(?i)facebookmail\.com|meta\.com`),
		CodePattern:  regexp.MustCompile(`(?:FB-|confirmation code[:\s]*)(\d{5,8})`),
		CodeGroup:    1,
	},
	{
		Name:         "Microsoft",
		SenderFilter: regexp.MustCompile(`(?i)microsoft\.com|outlook\.com`),
		CodePattern:  regexp.MustCompile(`(?i)(?:security code|code)[:\s]*(\d{4,8})`),
		CodeGroup:    1,
	},
	{
		Name:         "Amazon",
		SenderFilter: regexp.MustCompile(`(?i)amazon\.com`),
		CodePattern:  regexp.MustCompile(`(?i)(?:verification code|otp)[:\s]*(\d{4,8})`),
		CodeGroup:    1,
	},
	// Generic rules
	{
		Name:        "Generic numeric code",
		CodePattern: regexp.MustCompile(`(?i)(?:code|kode|код|pin|otp|verification|подтвержд)[:\s]*(\d{4,8})`),
		CodeGroup:   1,
	},
	{
		Name:        "Generic alphanumeric code",
		CodePattern: regexp.MustCompile(`(?i)(?:code|kode|код|pin|otp)[:\s]*([A-Za-z0-9]{4,10})`),
		CodeGroup:   1,
	},
	{
		Name:        "Standalone code line",
		CodePattern: regexp.MustCompile(`(?m)^\s*(\d{4,8})\s*$`),
		CodeGroup:   1,
	},
}
