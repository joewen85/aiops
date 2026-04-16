package auth

import "testing"

func TestJWTGenerateAndParse(t *testing.T) {
	manager := NewManager("unit-test-secret", 1)
	token, err := manager.GenerateToken(1, "tester", []string{"admin"}, "10")
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}
	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if claims.Username != "tester" {
		t.Fatalf("unexpected username: %s", claims.Username)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Fatalf("unexpected roles: %+v", claims.Roles)
	}
}

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("Secret@123")
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if err := ComparePassword(hash, "Secret@123"); err != nil {
		t.Fatalf("compare failed: %v", err)
	}
}
