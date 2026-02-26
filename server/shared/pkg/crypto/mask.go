package crypto

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// DataMasker provides utilities for masking sensitive data.
type DataMasker struct {
	// MaskChar is the character used for masking (default: '*')
	MaskChar rune

	// PreserveLength preserves the original length when masking
	PreserveLength bool
}

// NewDataMasker creates a new data masker with default settings.
func NewDataMasker() *DataMasker {
	return &DataMasker{
		MaskChar:       '*',
		PreserveLength: true,
	}
}

// MaskEmail masks an email address.
//
// Keeps first 2 characters of username and domain.
// Example: "john.doe@example.com" -> "jo******@ex*****.com"
func (dm *DataMasker) MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		// Invalid email, mask entire string
		return dm.maskString(email, 0, 0)
	}

	username := parts[0]
	domain := parts[1]

	// Mask username (keep first 2 chars)
	maskedUsername := dm.maskString(username, 2, 0)

	// Mask domain (keep first 2 chars and extension)
	domainParts := strings.Split(domain, ".")
	if len(domainParts) >= 2 {
		// Keep first 2 chars of domain and extension
		domainName := domainParts[0]
		extension := strings.Join(domainParts[1:], ".")
		maskedDomain := dm.maskString(domainName, 2, 0) + "." + extension
		return maskedUsername + "@" + maskedDomain
	}

	maskedDomain := dm.maskString(domain, 2, 0)
	return maskedUsername + "@" + maskedDomain
}

// MaskPhone masks a phone number.
//
// Keeps first 3 and last 2 digits.
// Example: "13812345678" -> "138*****78"
func (dm *DataMasker) MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}

	// Remove non-digit characters
	digits := regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")

	if len(digits) <= 5 {
		// Too short, mask all
		return dm.maskString(digits, 0, 0)
	}

	return dm.maskString(digits, 3, 2)
}

// MaskIDCard masks an ID card number.
//
// Keeps first 6 and last 4 digits.
// Example: "110101199001011234" -> "110101********1234"
func (dm *DataMasker) MaskIDCard(idCard string) string {
	if idCard == "" {
		return ""
	}

	if len(idCard) <= 10 {
		// Too short, mask middle
		return dm.maskString(idCard, 3, 3)
	}

	return dm.maskString(idCard, 6, 4)
}

// MaskBankCard masks a bank card number.
//
// Keeps first 4 and last 4 digits.
// Example: "6222021234567890123" -> "6222***********0123"
func (dm *DataMasker) MaskBankCard(cardNumber string) string {
	if cardNumber == "" {
		return ""
	}

	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(cardNumber, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if len(cleaned) <= 8 {
		return dm.maskString(cleaned, 2, 2)
	}

	return dm.maskString(cleaned, 4, 4)
}

// MaskName masks a person's name.
//
// Keeps first character, masks the rest.
// Example: "张三" -> "张*", "John Doe" -> "J*** D**"
func (dm *DataMasker) MaskName(name string) string {
	if name == "" {
		return ""
	}

	words := strings.Fields(name)
	if len(words) == 0 {
		return ""
	}

	var masked []string
	for _, word := range words {
		if utf8.RuneCountInString(word) <= 1 {
			masked = append(masked, word)
		} else {
			masked = append(masked, dm.maskString(word, 1, 0))
		}
	}

	return strings.Join(masked, " ")
}

// MaskPassword masks a password.
//
// Returns fixed length string of mask characters.
func (dm *DataMasker) MaskPassword(password string) string {
	if password == "" {
		return ""
	}

	if dm.PreserveLength {
		length := utf8.RuneCountInString(password)
		return strings.Repeat(string(dm.MaskChar), length)
	}

	// Fixed length
	return strings.Repeat(string(dm.MaskChar), 8)
}

// MaskToken masks an API token or secret.
//
// Shows prefix and masks the rest.
// Example: "sk_live_abc123def456" -> "sk_live_***"
func (dm *DataMasker) MaskToken(token string) string {
	if token == "" {
		return ""
	}

	parts := strings.Split(token, "_")
	if len(parts) > 1 {
		// Keep prefix (e.g., "sk_live")
		prefix := strings.Join(parts[:len(parts)-1], "_")
		return prefix + "_" + strings.Repeat(string(dm.MaskChar), 8)
	}

	// No prefix, show first 4 chars
	return dm.maskString(token, 4, 0)
}

// MaskIP masks an IP address.
//
// IPv4: Keeps first two octets, masks last two.
// Example: "192.168.1.1" -> "192.168.*.*"
func (dm *DataMasker) MaskIP(ip string) string {
	if ip == "" {
		return ""
	}

	// Check if IPv4
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return fmt.Sprintf("%s.%s.*.*", parts[0], parts[1])
	}

	// IPv6 or invalid, mask last half
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) >= 4 {
			visible := parts[:len(parts)/2]
			masked := strings.Repeat("*", len(parts)-len(visible))
			return strings.Join(visible, ":") + ":" + masked
		}
	}

	// Unknown format, mask half
	return dm.maskString(ip, len(ip)/2, 0)
}

// maskString masks a string, keeping the specified number of characters
// at the beginning and end.
func (dm *DataMasker) maskString(s string, keepStart, keepEnd int) string {
	if s == "" {
		return ""
	}

	length := utf8.RuneCountInString(s)

	// If keepStart + keepEnd >= length, return masked version
	if keepStart+keepEnd >= length {
		if keepStart >= length {
			return s
		}
		// Just keep start
		runes := []rune(s)
		result := string(runes[:keepStart])
		result += strings.Repeat(string(dm.MaskChar), length-keepStart)
		return result
	}

	runes := []rune(s)
	result := string(runes[:keepStart])
	result += strings.Repeat(string(dm.MaskChar), length-keepStart-keepEnd)
	result += string(runes[length-keepEnd:])

	return result
}

// AutoMask automatically detects the type of data and applies appropriate masking.
//
// It attempts to detect:
// - Email addresses
// - Phone numbers
// - ID cards (Chinese)
// - Bank cards
// - IP addresses
//
// If no pattern matches, it returns the input with middle section masked.
func (dm *DataMasker) AutoMask(data string) string {
	if data == "" {
		return ""
	}

	// Check email
	if isEmail(data) {
		return dm.MaskEmail(data)
	}

	// Check phone (Chinese format: 11 digits starting with 1)
	if isPhone(data) {
		return dm.MaskPhone(data)
	}

	// Check ID card (Chinese: 18 digits)
	if isIDCard(data) {
		return dm.MaskIDCard(data)
	}

	// Check bank card (13-19 digits)
	if isBankCard(data) {
		return dm.MaskBankCard(data)
	}

	// Check IP address
	if isIP(data) {
		return dm.MaskIP(data)
	}

	// Default: mask middle
	return dm.maskString(data, 3, 3)
}

// Helper functions for data type detection

func isEmail(s string) bool {
	// Simple email regex
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

func isPhone(s string) bool {
	// Chinese phone: 11 digits starting with 1
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

func isIDCard(s string) bool {
	// Chinese ID card: 18 characters (17 digits + 1 digit or X)
	pattern := `^\d{17}[\dXx]$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

func isBankCard(s string) bool {
	// Bank card: 13-19 digits
	pattern := `^\d{13,19}$`
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

func isIP(s string) bool {
	// Simple IP check (IPv4 or IPv6)
	return strings.Contains(s, ".") || strings.Contains(s, ":")
}
