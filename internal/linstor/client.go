package linstor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ntbc/linstor-mcp-server/internal/config"
	"github.com/ntbc/linstor-mcp-server/internal/kube"
)

type Client struct {
	baseURL     string
	httpClient  *http.Client
	portForward *kube.PortForward
}

type ControllerVersion struct {
	Version        string `json:"version"`
	GitHash        string `json:"git_hash"`
	BuildTime      string `json:"build_time"`
	RestAPIVersion string `json:"rest_api_version"`
}

type Node struct {
	Name             string             `json:"name"`
	Type             string             `json:"type"`
	ConnectionStatus string             `json:"connection_status"`
	UUID             string             `json:"uuid"`
	Props            map[string]string  `json:"props"`
	NetInterfaces    []NodeNetInterface `json:"net_interfaces"`
	StorageProviders []string           `json:"storage_providers"`
	ResourceLayers   []string           `json:"resource_layers"`
}

type NodeNetInterface struct {
	Name                    string `json:"name"`
	Address                 string `json:"address"`
	SatellitePort           int    `json:"satellite_port"`
	SatelliteEncryptionType string `json:"satellite_encryption_type"`
	IsActive                bool   `json:"is_active"`
	UUID                    string `json:"uuid"`
}

func New(ctx context.Context, cfg config.Config, kubeClient *kube.Client) (*Client, error) {
	baseURL := strings.TrimRight(cfg.ControllerURL, "/")
	var pf *kube.PortForward
	if baseURL == "" {
		if cfg.ControllerMode != "port-forward" {
			return nil, fmt.Errorf("controller_url is required when controller_mode=%s", cfg.ControllerMode)
		}
		var err error
		pf, err = kubeClient.StartPortForward(ctx, cfg.ControllerNamespace, cfg.ControllerService, 3371)
		if err != nil {
			return nil, err
		}
		baseURL = fmt.Sprintf("https://127.0.0.1:%d", pf.LocalPort)
	}

	caPEM, err := kubeClient.ReadSecretValue(ctx, cfg.ControllerNamespace, cfg.TLSCASecret, "ca.crt")
	if err != nil {
		caPEM, err = kubeClient.ReadSecretValue(ctx, cfg.ControllerNamespace, cfg.TLSCASecret, "tls.crt")
		if err != nil {
			if pf != nil {
				pf.Close()
			}
			return nil, fmt.Errorf("read CA secret: %w", err)
		}
	}
	certPEM, keyPEM, err := kubeClient.ReadTLSSecret(ctx, cfg.ControllerNamespace, cfg.TLSClientSecret)
	if err != nil {
		if pf != nil {
			pf.Close()
		}
		return nil, fmt.Errorf("read client TLS secret: %w", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		if pf != nil {
			pf.Close()
		}
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		if pf != nil {
			pf.Close()
		}
		return nil, fmt.Errorf("failed to append controller CA cert")
	}

	serverName := fmt.Sprintf("%s.%s.svc", cfg.ControllerService, cfg.ControllerNamespace)
	tlsConfig := &tls.Config{
		RootCAs:      roots,
		Certificates: []tls.Certificate{cert},
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		portForward: pf,
	}, nil
}

func (c *Client) Close() {
	if c.portForward != nil {
		c.portForward.Close()
	}
}

func (c *Client) ControllerVersion(ctx context.Context) (*ControllerVersion, error) {
	var out ControllerVersion
	if err := c.getJSON(ctx, "/v1/controller/version", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Nodes(ctx context.Context) ([]Node, error) {
	var out []Node
	if err := c.getJSON(ctx, "/v1/nodes", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("linstor API %s returned %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode linstor API %s: %w", path, err)
	}
	return nil
}
