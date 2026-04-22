package handler

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"devops-system/backend/internal/cloud"
	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/models"
)

var (
	errCloudProviderEmpty       = errors.New("provider cannot be empty")
	errCloudProviderUnsupported = errors.New("unsupported cloud provider")
)

func normalizeCloudProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func validateCloudCredentialInput(provider string, accessKey string, secretKey string) error {
	normalizedProvider := normalizeCloudProvider(provider)
	ak := strings.TrimSpace(accessKey)
	sk := strings.TrimSpace(secretKey)
	if strings.Contains(ak, "*") || strings.Contains(sk, "*") {
		return fmt.Errorf("credential looks masked, please input original accessKey/secretKey")
	}
	if normalizedProvider == "tencent" && ak != "" && !strings.HasPrefix(strings.ToUpper(ak), "AKID") {
		return fmt.Errorf("tencent accessKey should be SecretId (normally starts with AKID)")
	}
	return nil
}

func (h *Handler) cloudProviderByName(providerName string) (cloud.Provider, string, error) {
	normalized := normalizeCloudProvider(providerName)
	if normalized == "" {
		return nil, "", errCloudProviderEmpty
	}
	provider, exists := h.CloudProviders[normalized]
	if !exists {
		return nil, normalized, errCloudProviderUnsupported
	}
	return provider, normalized, nil
}

func (h *Handler) cloudProviderByAccount(account models.CloudAccount) (cloud.Provider, error) {
	provider, _, err := h.cloudProviderByName(account.Provider)
	return provider, err
}

func cloudProviderResolveAppError(err error) appErr.AppError {
	switch {
	case errors.Is(err, errCloudProviderEmpty):
		return appErr.New(3001, errCloudProviderEmpty.Error())
	case errors.Is(err, errCloudProviderUnsupported):
		return appErr.New(4003, errCloudProviderUnsupported.Error())
	default:
		return appErr.New(4003, defaultString(strings.TrimSpace(err.Error()), errCloudProviderUnsupported.Error()))
	}
}

func (h *Handler) cloudProviderExternalError(publicMessage string, err error) string {
	if err != nil {
		log.Printf("[cloud] %s detail=%v", publicMessage, err)
	}
	return publicMessage
}

func (h *Handler) cloudProviderExternalWarning(publicWarning string, err error) string {
	if err != nil {
		log.Printf("[cloud] %s detail=%v", publicWarning, err)
	}
	return publicWarning
}
