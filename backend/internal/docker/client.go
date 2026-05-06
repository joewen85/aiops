package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

const (
	DefaultTimeout       = 8 * time.Second
	DefaultActionTimeout = 60 * time.Second
)

type HostConfig struct {
	Endpoint  string
	TLSEnable bool
}

func ValidateEndpointSecurity(config HostConfig, env string) error {
	if _, err := NewClient(config); err != nil {
		return err
	}
	parsed, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || parsed.Scheme == "" {
		return errors.New("invalid docker endpoint")
	}
	normalizedEnv := strings.ToLower(strings.TrimSpace(env))
	if normalizedEnv == "" {
		normalizedEnv = "prod"
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "unix" {
		return nil
	}
	if normalizedEnv == "prod" {
		if scheme == "http" || (scheme == "tcp" && !config.TLSEnable) {
			return errors.New("prod docker endpoint requires TLS or unix socket")
		}
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return errors.New("docker endpoint host is required")
	}
	if host == "metadata.google.internal" || strings.HasSuffix(host, ".metadata.google.internal") {
		return errors.New("metadata endpoint is not allowed")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, lookupErr := net.LookupIP(host)
		if lookupErr == nil && len(ips) > 0 {
			ip = ips[0]
		}
	}
	if ip == nil {
		return nil
	}
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return errors.New("unsafe docker endpoint address is not allowed")
	}
	if normalizedEnv == "prod" && ip.IsLoopback() {
		return errors.New("prod docker endpoint cannot use loopback tcp/http address")
	}
	return nil
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type PingResult struct {
	OK         bool              `json:"ok"`
	APIVersion string            `json:"apiVersion,omitempty"`
	OSType     string            `json:"osType,omitempty"`
	RawHeaders map[string]string `json:"rawHeaders,omitempty"`
}

func NewClient(config HostConfig) (*Client, error) {
	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		return nil, errors.New("docker endpoint is required")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	baseURL, err := normalizeEndpoint(endpoint, config.TLSEnable, transport)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
		},
	}, nil
}

func (c *Client) Ping(ctx context.Context) (PingResult, error) {
	resp, err := c.do(ctx, http.MethodGet, "/_ping", nil)
	if err != nil {
		return PingResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PingResult{}, responseError(resp)
	}
	result := PingResult{
		OK:         true,
		APIVersion: resp.Header.Get("API-Version"),
		OSType:     resp.Header.Get("OSType"),
		RawHeaders: map[string]string{},
	}
	for key, values := range resp.Header {
		if len(values) > 0 {
			result.RawHeaders[key] = values[0]
		}
	}
	return result, nil
}

func (c *Client) Version(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.getJSON(ctx, "/version", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListContainers(ctx context.Context) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	if err := c.getJSON(ctx, "/containers/json?all=1", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListImages(ctx context.Context) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	if err := c.getJSON(ctx, "/images/json", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListNetworks(ctx context.Context) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	if err := c.getJSON(ctx, "/networks", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListVolumes(ctx context.Context) ([]map[string]interface{}, error) {
	var payload struct {
		Volumes []map[string]interface{} `json:"Volumes"`
	}
	if err := c.getJSON(ctx, "/volumes", &payload); err != nil {
		return nil, err
	}
	if payload.Volumes == nil {
		return []map[string]interface{}{}, nil
	}
	return payload.Volumes, nil
}

func (c *Client) ContainerAction(ctx context.Context, containerID string, action string) error {
	containerID = strings.TrimSpace(containerID)
	action = strings.ToLower(strings.TrimSpace(action))
	if containerID == "" {
		return errors.New("container id is required")
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return fmt.Errorf("unsupported container action %q", action)
	}
	resp, err := c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(containerID)+"/"+action, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	return nil
}

func (c *Client) RemoveImage(ctx context.Context, imageID string) error {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return errors.New("image id is required")
	}
	resp, err := c.do(ctx, http.MethodDelete, "/images/"+url.PathEscape(imageID)+"?force=0&noprune=0", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	return nil
}

func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	networkID = strings.TrimSpace(networkID)
	if networkID == "" {
		return errors.New("network id is required")
	}
	resp, err := c.do(ctx, http.MethodDelete, "/networks/"+url.PathEscape(networkID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	return nil
}

func (c *Client) RemoveVolume(ctx context.Context, volumeName string) error {
	volumeName = strings.TrimSpace(volumeName)
	if volumeName == "" {
		return errors.New("volume name is required")
	}
	resp, err := c.do(ctx, http.MethodDelete, "/volumes/"+url.PathEscape(volumeName)+"?force=0", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	return nil
}

type ComposeRunner struct {
	Host HostConfig
}

func (r ComposeRunner) Run(ctx context.Context, projectName string, content string, action string) (string, error) {
	projectName = sanitizeComposeProjectName(projectName)
	content = strings.TrimSpace(content)
	action = strings.ToLower(strings.TrimSpace(action))
	if projectName == "" {
		return "", errors.New("compose project name is required")
	}
	if content == "" {
		return "", errors.New("compose content is required")
	}
	args, err := composeArgsForAction(action)
	if err != nil {
		return "", err
	}
	tmpFile, err := os.CreateTemp("", "aiops-compose-*.yaml")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmpFile.WriteString(content + "\n"); err != nil {
		_ = tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}
	baseArgs := []string{"compose", "-p", projectName, "-f", tmpPath}
	cmdArgs := append(baseArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Env = composeEnv(os.Environ(), r.Host)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker compose %s failed: %w output=%s", action, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func (c *Client) getJSON(ctx context.Context, requestPath string, out interface{}) error {
	resp, err := c.do(ctx, http.MethodGet, requestPath, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) do(ctx context.Context, method string, requestPath string, body interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+requestPath, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

func normalizeEndpoint(endpoint string, tlsEnable bool, transport *http.Transport) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" {
		return "", errors.New("invalid docker endpoint")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		if parsed.Host == "" {
			return "", errors.New("docker endpoint host is required")
		}
		return endpoint, nil
	case "tcp":
		if parsed.Host == "" {
			return "", errors.New("docker tcp endpoint host is required")
		}
		scheme := "http"
		if tlsEnable {
			scheme = "https"
		}
		return scheme + "://" + parsed.Host, nil
	case "unix":
		socketPath := parsed.Path
		if socketPath == "" {
			socketPath = parsed.Host
		}
		socketPath = path.Clean(socketPath)
		if socketPath != "/var/run/docker.sock" {
			return "", errors.New("only unix:///var/run/docker.sock is allowed")
		}
		transport.DialContext = func(ctx context.Context, network string, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		}
		return "http://docker", nil
	default:
		return "", errors.New("unsupported docker endpoint scheme")
	}
}

func composeArgsForAction(action string) ([]string, error) {
	switch action {
	case "validate":
		return []string{"config", "--quiet"}, nil
	case "deploy", "up":
		return []string{"up", "-d", "--remove-orphans"}, nil
	case "down":
		return []string{"down", "--remove-orphans"}, nil
	case "restart":
		return []string{"restart"}, nil
	default:
		return nil, fmt.Errorf("unsupported compose action %q", action)
	}
}

func composeEnv(base []string, host HostConfig) []string {
	env := append([]string{}, base...)
	endpoint := strings.TrimSpace(host.Endpoint)
	if strings.HasPrefix(endpoint, "tcp://") || strings.HasPrefix(endpoint, "unix://") || strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		env = append(env, "DOCKER_HOST="+endpoint)
	}
	if host.TLSEnable {
		env = append(env, "DOCKER_TLS_VERIFY=1")
	}
	return env
}

func sanitizeComposeProjectName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var builder strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-_")
}

func responseError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(raw))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("docker api failed: status=%d message=%s", resp.StatusCode, message)
}
