package handler

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"devops-system/backend/internal/cloud"
	"devops-system/backend/internal/models"
)

const cloudCredentialEncryptedPrefix = "enc:v1:"

func (h *Handler) cloudCredentialCipherKey() ([]byte, error) {
	configuredSeed := strings.TrimSpace(h.Config.CloudCredentialEncryptKey)
	jwtSeed := strings.TrimSpace(h.Config.JWTSecret)
	seed := configuredSeed
	if seed == "" {
		seed = jwtSeed
	}
	if seed == "" {
		seed = "devops-cloud-credential-seed"
	}
	if strings.EqualFold(strings.TrimSpace(h.Config.AppEnv), "production") {
		if configuredSeed == "" && (jwtSeed == "" || jwtSeed == "change-me") {
			return nil, fmt.Errorf("cloud credential encryption seed is not secure in production, please set CLOUD_CREDENTIAL_ENCRYPT_KEY")
		}
	}
	sum := sha256.Sum256([]byte(seed))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return key, nil
}

func (h *Handler) encryptCloudCredential(plainText string) (string, error) {
	raw := strings.TrimSpace(plainText)
	if raw == "" {
		return "", nil
	}
	key, err := h.cloudCredentialCipherKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("init aes cipher failed: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("init gcm failed: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce failed: %w", err)
	}
	cipherText := aead.Seal(nil, nonce, []byte(raw), nil)
	payload := append(nonce, cipherText...)
	return cloudCredentialEncryptedPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func (h *Handler) decryptCloudCredential(cipherText string) (string, error) {
	value := strings.TrimSpace(cipherText)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, cloudCredentialEncryptedPrefix) {
		return value, nil
	}
	rawPayload := strings.TrimPrefix(value, cloudCredentialEncryptedPrefix)
	payload, err := base64.StdEncoding.DecodeString(rawPayload)
	if err != nil {
		return "", fmt.Errorf("decode encrypted credential failed: %w", err)
	}
	key, err := h.cloudCredentialCipherKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("init aes cipher failed: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("init gcm failed: %w", err)
	}
	nonceSize := aead.NonceSize()
	if len(payload) < nonceSize {
		return "", fmt.Errorf("encrypted credential payload invalid")
	}
	nonce := payload[:nonceSize]
	data := payload[nonceSize:]
	plain, err := aead.Open(nil, nonce, data, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt cloud credential failed")
	}
	return strings.TrimSpace(string(plain)), nil
}

func (h *Handler) cloudCredentials(account models.CloudAccount) (cloud.Credentials, error) {
	accessKey, err := h.decryptCloudCredential(account.AccessKey)
	if err != nil {
		return cloud.Credentials{}, err
	}
	secretKey, err := h.decryptCloudCredential(account.SecretKey)
	if err != nil {
		return cloud.Credentials{}, err
	}
	return cloud.Credentials{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    account.Region,
	}, nil
}

func (h *Handler) migrateCloudCredentialsIfPlain(account *models.CloudAccount, plainCred cloud.Credentials) error {
	if account == nil {
		return nil
	}
	accessKeyEncrypted := strings.HasPrefix(strings.TrimSpace(account.AccessKey), cloudCredentialEncryptedPrefix)
	secretKeyEncrypted := strings.HasPrefix(strings.TrimSpace(account.SecretKey), cloudCredentialEncryptedPrefix)
	if accessKeyEncrypted && secretKeyEncrypted {
		return nil
	}
	encryptedAccessKey, err := h.encryptCloudCredential(plainCred.AccessKey)
	if err != nil {
		return err
	}
	encryptedSecretKey, err := h.encryptCloudCredential(plainCred.SecretKey)
	if err != nil {
		return err
	}
	if err := h.DB.Model(&models.CloudAccount{}).Where("id = ?", account.ID).Updates(map[string]interface{}{
		"access_key": encryptedAccessKey,
		"secret_key": encryptedSecretKey,
	}).Error; err != nil {
		return err
	}
	account.AccessKey = encryptedAccessKey
	account.SecretKey = encryptedSecretKey
	return nil
}
