package crypto

import (
	"testing"
)

func TestDataMasker_MaskEmail(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"john.doe@example.com", "jo******@ex*****.com"},
		{"a@b.com", "a@b.com"}, // too short to mask
		{"test@domain.co.uk", "te**@do****.co.uk"},
		{"", ""},
		{"invalid-email", "*************"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskEmail(tt.input)
			if result != tt.expected {
				t.Errorf("MaskEmail(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskPhone(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"13812345678", "138******78"},
		{"138-1234-5678", "138******78"},
		{"(138) 1234-5678", "138******78"},
		{"12345", "*****"}, // too short
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskPhone(tt.input)
			if result != tt.expected {
				t.Errorf("MaskPhone(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskIDCard(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"110101199001011234", "110101********1234"},
		{"12345678", "123**678"}, // too short, fallback
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskIDCard(tt.input)
			if result != tt.expected {
				t.Errorf("MaskIDCard(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskBankCard(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"6222021234567890123", "6222***********0123"},
		{"6222 0212 3456 7890 123", "6222***********0123"},
		{"12345678", "12****78"},
		{"123456", "12**56"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskBankCard(tt.input)
			if result != tt.expected {
				t.Errorf("MaskBankCard(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskName(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"张三", "张*"},
		{"John Doe", "J*** D**"},
		{"Alice", "A****"},
		{"A", "A"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskName(tt.input)
			if result != tt.expected {
				t.Errorf("MaskName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskPassword(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"password123", "***********"},
		{"abc", "***"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskPassword(tt.input)
			if result != tt.expected {
				t.Errorf("MaskPassword(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskToken(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"sk_live_abc123def456", "sk_live_********"},
		{"pk_test_xyz789", "pk_test_********"},
		{"simple_token", "simple_********"},
		{"noprefix", "nopr****"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskToken(tt.input)
			if result != tt.expected {
				t.Errorf("MaskToken(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_MaskIP(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1", "192.168.*.*"},
		{"10.0.0.1", "10.0.*.*"},
		{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:0db8:85a3:0000:****"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dm.MaskIP(tt.input)
			if result != tt.expected {
				t.Errorf("MaskIP(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_AutoMask(t *testing.T) {
	dm := NewDataMasker()

	tests := []struct {
		name     string
		input    string
		dataType string // for documentation
	}{
		{"email", "john@example.com", "email"},
		{"phone", "13812345678", "phone"},
		{"id card", "110101199001011234", "id_card"},
		{"bank card", "6222021234567890123", "bank_card"},
		{"ip", "192.168.1.1", "ip"},
		{"unknown", "some random text", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dm.AutoMask(tt.input)

			// Should not be empty
			if result == "" && tt.input != "" {
				t.Error("AutoMask returned empty string")
			}

			// Should contain mask characters
			if tt.input != "" && result == tt.input {
				t.Error("AutoMask didn't mask the input")
			}
		})
	}
}

func TestDataMasker_CustomMaskChar(t *testing.T) {
	dm := &DataMasker{
		MaskChar:       'X',
		PreserveLength: true,
	}

	result := dm.MaskEmail("test@example.com")
	if result[2] != 'X' {
		t.Errorf("expected mask character 'X', but got '%c'", result[2])
	}
}

func TestDataMasker_PreserveLengthFalse(t *testing.T) {
	dm := &DataMasker{
		MaskChar:       '*',
		PreserveLength: false,
	}

	password := "very-long-password-123456"
	result := dm.MaskPassword(password)

	// Should be fixed length
	if len(result) != 8 {
		t.Errorf("expected fixed length 8, got %d", len(result))
	}
}

func TestIsEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"test@example.com", true},
		{"user+tag@domain.co.uk", true},
		{"invalid", false},
		{"@domain.com", false},
		{"user@", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isEmail(tt.input)
			if result != tt.expected {
				t.Errorf("isEmail(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"13812345678", true},
		{"15912345678", true},
		{"12345678901", false}, // doesn't start with 13-19
		{"138123456", false},   // too short
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isPhone(tt.input)
			if result != tt.expected {
				t.Errorf("isPhone(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsIDCard(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"110101199001011234", true},
		{"11010119900101123X", true},
		{"1101011990010112", false},   // too short
		{"110101199001011234a", false}, // extra char
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isIDCard(tt.input)
			if result != tt.expected {
				t.Errorf("isIDCard(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsBankCard(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"6222021234567890123", true},
		{"1234567890123", true},
		{"12345678901234567890", false}, // too long
		{"123456789012", false},         // too short
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isBankCard(tt.input)
			if result != tt.expected {
				t.Errorf("isBankCard(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsIP(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"192.168.1.1", true},
		{"2001:0db8:85a3::7334", true},
		{"not-an-ip", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isIP(tt.input)
			if result != tt.expected {
				t.Errorf("isIP(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDataMasker_EdgeCases(t *testing.T) {
	dm := NewDataMasker()

	// Empty strings
	if dm.MaskEmail("") != "" {
		t.Error("expected empty result for empty email")
	}

	// Very short inputs
	if dm.MaskPhone("123") != "***" {
		t.Error("short phone not masked correctly")
	}

	// Unicode characters
	result := dm.MaskName("李明")
	if result[0:3] != "李" { // First character should be preserved (UTF-8 takes 3 bytes)
		t.Error("unicode name not masked correctly")
	}
}

func TestDataMasker_ConcurrentMasking(t *testing.T) {
	dm := NewDataMasker()

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			_ = dm.MaskEmail("test@example.com")
			_ = dm.MaskPhone("13812345678")
			_ = dm.AutoMask("some data")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

// Benchmarks

func BenchmarkMaskEmail(b *testing.B) {
	dm := NewDataMasker()
	email := "john.doe@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dm.MaskEmail(email)
	}
}

func BenchmarkMaskPhone(b *testing.B) {
	dm := NewDataMasker()
	phone := "13812345678"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dm.MaskPhone(phone)
	}
}

func BenchmarkMaskIDCard(b *testing.B) {
	dm := NewDataMasker()
	idCard := "110101199001011234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dm.MaskIDCard(idCard)
	}
}

func BenchmarkAutoMask(b *testing.B) {
	dm := NewDataMasker()
	data := "test@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dm.AutoMask(data)
	}
}
