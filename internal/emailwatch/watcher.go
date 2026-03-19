package emailwatch

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"expense-bot/internal/domain"
	"expense-bot/internal/storage"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Watcher manages IMAP IDLE connections for all active email accounts.
type Watcher struct {
	store         storage.Storage
	notifier      *TelegramNotifier
	encryptionKey string
	idleTimeout   time.Duration
	maxBackoff    time.Duration

	mu       sync.Mutex
	watchers map[int64]context.CancelFunc // account ID -> cancel
}

// NewWatcher creates a new email watcher.
func NewWatcher(store storage.Storage, notifier *TelegramNotifier, encryptionKey string, idleTimeout, maxBackoff time.Duration) *Watcher {
	return &Watcher{
		store:         store,
		notifier:      notifier,
		encryptionKey: encryptionKey,
		idleTimeout:   idleTimeout,
		maxBackoff:    maxBackoff,
		watchers:      make(map[int64]context.CancelFunc),
	}
}

// Start loads all active email accounts and starts watching them.
func (w *Watcher) Start(ctx context.Context) error {
	accounts, err := w.store.GetActiveEmailAccounts(ctx)
	if err != nil {
		return fmt.Errorf("watcher.Start: %w", err)
	}

	slog.Info("starting email watcher", "accounts", len(accounts))

	for _, acc := range accounts {
		w.StartAccount(ctx, acc)
	}

	// Periodically clean old codes
	go w.cleanupLoop(ctx)

	return nil
}

// StartAccount starts watching a single email account.
func (w *Watcher) StartAccount(ctx context.Context, acc *domain.EmailAccount) {
	w.mu.Lock()
	if cancel, ok := w.watchers[acc.ID]; ok {
		cancel()
	}
	accCtx, cancel := context.WithCancel(ctx)
	w.watchers[acc.ID] = cancel
	w.mu.Unlock()

	go w.watchAccount(accCtx, acc)
}

// StopAccount stops watching a single email account.
func (w *Watcher) StopAccount(accountID int64) {
	w.mu.Lock()
	if cancel, ok := w.watchers[accountID]; ok {
		cancel()
		delete(w.watchers, accountID)
	}
	w.mu.Unlock()
}

// Stop stops all watchers.
func (w *Watcher) Stop() {
	w.mu.Lock()
	for id, cancel := range w.watchers {
		cancel()
		delete(w.watchers, id)
	}
	w.mu.Unlock()
}

func (w *Watcher) watchAccount(ctx context.Context, acc *domain.EmailAccount) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := w.connectAndWatch(ctx, acc)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("IMAP connection error",
				"error", err,
				"account_id", acc.ID,
				"email", acc.Email,
				"retry_in", backoff,
			)

			errStr := err.Error()
			w.store.UpdateEmailAccountStatus(ctx, acc.ID, nil, &errStr)

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			// Exponential backoff
			backoff *= 2
			if backoff > w.maxBackoff {
				backoff = w.maxBackoff
			}
		} else {
			backoff = time.Second // reset on success
		}
	}
}

func (w *Watcher) connectAndWatch(ctx context.Context, acc *domain.EmailAccount) error {
	password, err := Decrypt(acc.PasswordEnc, w.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt password: %w", err)
	}

	// Connect to IMAP server
	c, err := imapclient.DialTLS(acc.IMAPServer, &imapclient.Options{
		TLSConfig: &tls.Config{},
	})
	if err != nil {
		return fmt.Errorf("dial TLS %s: %w", acc.IMAPServer, err)
	}
	defer c.Close()

	// Login
	if err := c.Login(acc.Email, password).Wait(); err != nil {
		return fmt.Errorf("login %s: %w", acc.Email, err)
	}
	defer c.Logout()

	// Select INBOX
	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		return fmt.Errorf("select INBOX: %w", err)
	}

	// Update status
	now := time.Now().Format(time.RFC3339)
	w.store.UpdateEmailAccountStatus(ctx, acc.ID, &now, nil)

	slog.Info("IMAP connected", "email", acc.Email)

	// Check for any unseen messages that arrived before we connected
	w.processNewMessages(ctx, c, acc)

	// IDLE loop — wait for new messages, then process
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		idleCmd, err := c.Idle()
		if err != nil {
			return fmt.Errorf("start IDLE: %w", err)
		}

		// Wait for IDLE timeout (server may also interrupt on new mail)
		timer := time.NewTimer(w.idleTimeout)
		select {
		case <-ctx.Done():
			timer.Stop()
			idleCmd.Close()
			return nil
		case <-timer.C:
		}

		// Close IDLE to resume normal commands
		if err := idleCmd.Close(); err != nil {
			return fmt.Errorf("close IDLE: %w", err)
		}

		// Process any new messages that arrived during IDLE
		w.processNewMessages(ctx, c, acc)
	}
}

func (w *Watcher) processNewMessages(ctx context.Context, c *imapclient.Client, acc *domain.EmailAccount) {
	// Search for unseen messages
	criteria := &imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}

	searchData, err := c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		slog.Error("search unseen failed", "error", err, "email", acc.Email)
		return
	}

	if len(searchData.AllUIDs()) == 0 {
		return
	}

	// Fetch messages
	fetchOpts := &imap.FetchOptions{
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	seqSet := imap.UIDSetNum(searchData.AllUIDs()...)
	fetchCmd := c.Fetch(seqSet, fetchOpts)
	defer fetchCmd.Close()

	for {
		msgData := fetchCmd.Next()
		if msgData == nil {
			break
		}

		buf, err := msgData.Collect()
		if err != nil {
			slog.Warn("collect message failed", "error", err, "email", acc.Email)
			continue
		}

		for _, section := range buf.BodySection {
			if len(section.Bytes) == 0 {
				continue
			}

			result, err := ParseEmail(strings.NewReader(string(section.Bytes)))
			if err != nil {
				slog.Warn("parse email failed", "error", err, "email", acc.Email)
				continue
			}
			if result == nil {
				continue
			}

			// Check dedup
			exists, _ := w.store.CodeExistsByBodyHash(ctx, result.BodyHash)
			if exists {
				continue
			}

			// Send notification
			msgID, err := w.notifier.NotifyCode(acc.UserID, acc.Email, result, time.Now())
			if err != nil {
				slog.Error("notify code failed", "error", err)
				continue
			}

			// Save to DB
			_, err = w.store.CreateEmailCode(ctx, &domain.EmailCode{
				EmailAccountID: acc.ID,
				UserID:         acc.UserID,
				Sender:         result.Sender,
				Subject:        result.Subject,
				Code:           result.Code,
				RuleName:       result.RuleName,
				RawBodyHash:    result.BodyHash,
				TgMessageID:    msgID,
				ReceivedAt:     time.Now(),
			})
			if err != nil {
				slog.Error("save email code failed", "error", err)
			}
		}
	}
}

func (w *Watcher) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted, err := w.store.DeleteOldCodes(ctx)
			if err != nil {
				slog.Error("cleanup old codes failed", "error", err)
			} else if deleted > 0 {
				slog.Info("cleaned up old codes", "deleted", deleted)
			}
		}
	}
}

// ValidateIMAPConnection tests that the IMAP credentials are valid.
func ValidateIMAPConnection(server, email, password string) error {
	if !strings.Contains(server, ":") {
		server += ":993"
	}

	c, err := imapclient.DialTLS(server, &imapclient.Options{
		TLSConfig: &tls.Config{},
	})
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", server, err)
	}
	defer c.Close()

	if err := c.Login(email, password).Wait(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	c.Logout()

	return nil
}
