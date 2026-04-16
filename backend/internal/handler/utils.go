package handler

import (
	"strconv"

	"devops-system/backend/internal/auth"
)

func comparePassword(hash string, password string) error {
	return auth.ComparePassword(hash, password)
}

func hashPassword(password string) (string, error) {
	return auth.HashPassword(password)
}

func toStringID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
