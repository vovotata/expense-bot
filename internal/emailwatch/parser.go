package emailwatch

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// ParseResult holds the result of parsing an email for verification codes.
type ParseResult struct {
	Sender   string
	Subject  string
	Code     string
	RuleName string
	BodyHash string
}

// ParseEmail tries to extract a verification code from the given raw email.
func ParseEmail(rawEmail io.Reader) (*ParseResult, error) {
	msg, err := mail.ReadMessage(rawEmail)
	if err != nil {
		return nil, fmt.Errorf("parser.ParseEmail: %w", err)
	}

	from := msg.Header.Get("From")
	subject := msg.Header.Get("Subject")
	body, err := extractBody(msg)
	if err != nil {
		return nil, fmt.Errorf("parser.ParseEmail: extract body: %w", err)
	}

	bodyHash := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))

	for _, rule := range DefaultRules {
		if rule.SenderFilter != nil && !rule.SenderFilter.MatchString(from) {
			continue
		}
		if rule.SubjectFilter != nil && !rule.SubjectFilter.MatchString(subject) {
			continue
		}

		matches := rule.CodePattern.FindStringSubmatch(body)
		if len(matches) > rule.CodeGroup {
			return &ParseResult{
				Sender:   from,
				Subject:  subject,
				Code:     matches[rule.CodeGroup],
				RuleName: rule.Name,
				BodyHash: bodyHash,
			}, nil
		}

		// Also try subject line
		matches = rule.CodePattern.FindStringSubmatch(subject)
		if len(matches) > rule.CodeGroup {
			return &ParseResult{
				Sender:   from,
				Subject:  subject,
				Code:     matches[rule.CodeGroup],
				RuleName: rule.Name,
				BodyHash: bodyHash,
			}, nil
		}
	}

	return nil, nil // no code found
}

// extractBody extracts text content from an email, handling multipart.
func extractBody(msg *mail.Message) (string, error) {
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Fall back to reading body as-is
		b, _ := io.ReadAll(msg.Body)
		return string(b), nil
	}

	if strings.HasPrefix(mediaType, "text/") {
		b, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", err
		}
		return stripHTML(string(b)), nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		return extractMultipart(msg.Body, params["boundary"])
	}

	b, _ := io.ReadAll(msg.Body)
	return string(b), nil
}

func extractMultipart(body io.Reader, boundary string) (string, error) {
	reader := multipart.NewReader(body, boundary)
	var textParts []string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return strings.Join(textParts, "\n"), nil
		}

		ct := part.Header.Get("Content-Type")
		if ct == "" {
			ct = "text/plain"
		}

		mt, partParams, _ := mime.ParseMediaType(ct)

		if strings.HasPrefix(mt, "multipart/") {
			nested, _ := extractMultipart(part, partParams["boundary"])
			textParts = append(textParts, nested)
			continue
		}

		if strings.HasPrefix(mt, "text/") {
			b, _ := io.ReadAll(part)
			content := string(b)
			if mt == "text/html" {
				content = stripHTML(content)
			}
			textParts = append(textParts, content)
		}
	}

	return strings.Join(textParts, "\n"), nil
}

// stripHTML removes HTML tags in a simple way.
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			result.WriteRune(' ')
		case !inTag:
			result.WriteRune(r)
		}
	}
	return result.String()
}
