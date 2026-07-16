package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidPhone = errors.New("invalid phone number")
	ErrInvalidOTP   = errors.New("invalid or expired otp")
)

var ugandaPhone = regexp.MustCompile(`^\+256[0-9]{9}$`)

type SMSProvider interface {
	SendOTP(ctx context.Context, phone, code string) error
}

type ConsoleSMS struct{}

func (ConsoleSMS) SendOTP(_ context.Context, phone, code string) error {
	log.Printf("[auth] OTP for %s: %s", phone, code)
	return nil
}

type Service struct {
	repo     *Repository
	tokens   *TokenService
	sms      SMSProvider
	otpTTL   time.Duration
}

func NewService(repo *Repository, tokens *TokenService, sms SMSProvider, otpTTL time.Duration) *Service {
	return &Service{repo: repo, tokens: tokens, sms: sms, otpTTL: otpTTL}
}

func NormalizePhone(raw string) (string, error) {
	phone := strings.TrimSpace(raw)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	switch {
	case strings.HasPrefix(phone, "+256"):
		// ok
	case strings.HasPrefix(phone, "256"):
		phone = "+" + phone
	case strings.HasPrefix(phone, "0") && len(phone) == 10:
		phone = "+256" + phone[1:]
	default:
		return "", ErrInvalidPhone
	}

	if !ugandaPhone.MatchString(phone) {
		return "", ErrInvalidPhone
	}
	return phone, nil
}

func (s *Service) RequestOTP(ctx context.Context, rawPhone string) error {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return err
	}

	code, err := GenerateOTP(6)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	expiresAt := time.Now().Add(s.otpTTL)
	if err := s.repo.SaveOTP(ctx, phone, code, expiresAt); err != nil {
		return fmt.Errorf("save otp: %w", err)
	}

	if err := s.sms.SendOTP(ctx, phone, code); err != nil {
		return fmt.Errorf("send otp: %w", err)
	}
	return nil
}

func (s *Service) VerifyOTP(ctx context.Context, rawPhone, code, requestedRole string) (*User, string, time.Time, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return nil, "", time.Time{}, ErrInvalidOTP
	}

	ok, err := s.repo.VerifyOTP(ctx, phone, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", time.Time{}, ErrInvalidOTP
		}
		return nil, "", time.Time{}, err
	}
	if !ok {
		return nil, "", time.Time{}, ErrInvalidOTP
	}

	user, err := s.repo.GetUserByPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		role := defaultRole(requestedRole)
		user, err = s.repo.CreateUser(ctx, phone, role)
		if err != nil {
			return nil, "", time.Time{}, err
		}
	} else if err != nil {
		return nil, "", time.Time{}, err
	} else {
		user, err = s.applyRequestedRole(ctx, user, requestedRole)
		if err != nil {
			return nil, "", time.Time{}, err
		}
	}

	token, expiresAt, err := s.tokens.Issue(user)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	return user, token, expiresAt, nil
}

func defaultRole(requested string) string {
	switch requested {
	case "driver", "passenger", "admin":
		return requested
	default:
		return "passenger"
	}
}

// applyRequestedRole lets the same phone use driver or passenger apps before
// operator registration locks the account as a driver.
func (s *Service) applyRequestedRole(ctx context.Context, user *User, requestedRole string) (*User, error) {
	if user.Role == "admin" {
		return user, nil
	}

	desired := defaultRole(requestedRole)
	if desired == user.Role || desired == "admin" {
		return user, nil
	}

	hasOperator, err := s.repo.HasOperatorProfile(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if hasOperator {
		return user, nil
	}

	return s.repo.SetUserRole(ctx, user.ID, desired)
}

func (s *Service) Me(ctx context.Context, userID string) (*User, error) {
	return s.repo.GetUserByID(ctx, userID)
}
