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
		Name:          "Microsoft",
		SenderFilter:  regexp.MustCompile(`(?i)microsoft\.com|outlook\.com`),
		CodePattern:   regexp.MustCompile(`(?i)(?:security code|code)[:\s]*(\d{4,8})`),
		CodeGroup:     1,
	},
	{
		Name:         "Amazon",
		SenderFilter: regexp.MustCompile(`(?i)amazon\.com`),
		CodePattern:  regexp.MustCompile(`(?i)(?:verification code|otp)[:\s]*(\d{4,8})`),
		CodeGroup:    1,
	},
	{
		Name:        "Generic numeric code",
		CodePattern: regexp.MustCompile(`(?i)(?:code|код|pin|otp|verification)[:\s]*(\d{4,8})`),
		CodeGroup:   1,
	},
	{
		Name:        "Generic alphanumeric code",
		CodePattern: regexp.MustCompile(`(?i)(?:code|код|pin)[:\s]*([A-Z0-9]{4,10})`),
		CodeGroup:   1,
	},
}
