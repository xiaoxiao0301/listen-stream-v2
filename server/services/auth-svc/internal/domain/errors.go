package domain

import "errors"

// User 相关错误
var (
	ErrInvalidUserID       = errors.New("invalid user ID")
	ErrInvalidPhone        = errors.New("invalid phone number")
	ErrInvalidPhoneFormat  = errors.New("invalid phone number format (must be 11 digits)")
	ErrInvalidTokenVersion = errors.New("invalid token version")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserInactive        = errors.New("user is inactive")
)

// Device 相关错误
var (
	ErrInvalidDeviceID     = errors.New("invalid device ID")
	ErrInvalidDeviceName   = errors.New("invalid device name")
	ErrInvalidFingerprint  = errors.New("invalid device fingerprint")
	ErrInvalidPlatform     = errors.New("invalid platform")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrMaxDevicesExceeded  = errors.New("maximum number of devices exceeded")
	ErrSuspiciousLogin     = errors.New("suspicious login detected")
	ErrFingerprintMismatch = errors.New("device fingerprint mismatch")
)

// SMS 相关错误
var (
	ErrInvalidSMSID         = errors.New("invalid SMS ID")
	ErrInvalidSMSCode       = errors.New("invalid SMS code")
	ErrInvalidSMSCodeLength = errors.New("invalid SMS code length (must be 6 digits)")
	ErrSMSCodeExpired       = errors.New("SMS code expired")
	ErrSMSCodeAlreadyUsed   = errors.New("SMS code already used")
	ErrSMSCodeInvalid       = errors.New("SMS verification code is invalid")
	ErrSMSTooFrequent       = errors.New("SMS sent too frequently, please wait")
	ErrSMSSendFailed        = errors.New("failed to send SMS")
)

// SMSRecord 相关错误
var (
	ErrInvalidSMSRecordID = errors.New("invalid SMS record ID")
	ErrInvalidSMSProvider = errors.New("invalid SMS provider")
)
