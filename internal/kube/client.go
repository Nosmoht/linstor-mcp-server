package kube

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

var (
	GVRLinstorCluster = schema.GroupVersionResource{Group: "piraeus.io", Version: "v1", Resource: "linstorclusters"}
	GVRSatelliteCfg   = schema.GroupVersionResource{Group: "piraeus.io", Version: "v1", Resource: "linstorsatelliteconfigurations"}
	GVRNodeConnection = schema.GroupVersionResource{Group: "piraeus.io", Version: "v1", Resource: "linstornodeconnections"}

	GVRInternalNodes        = schema.GroupVersionResource{Group: "internal.linstor.linbit.com", Version: "v1-15-0", Resource: "nodes"}
	GVRInternalNodeStorPool = schema.GroupVersionResource{Group: "internal.linstor.linbit.com", Version: "v1-15-0", Resource: "nodestorpool"}
	GVRInternalResourceDef  = schema.GroupVersionResource{Group: "internal.linstor.linbit.com", Version: "v1-15-0", Resource: "resourcedefinitions"}
	GVRInternalResources    = schema.GroupVersionResource{Group: "internal.linstor.linbit.com", Version: "v1-15-0", Resource: "resources"}
)

type Client struct {
	RESTConfig *rest.Config
	Core       kubernetes.Interface
	Dynamic    dynamic.Interface
	Discovery  discovery.DiscoveryInterface
	CurrentCtx string
}

type PortForward struct {
	LocalPort int
	stopCh    chan struct{}
	readyCh   chan struct{}
}

func New(ctxName string) (*Client, error) {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	if ctxName != "" {
		overrides.CurrentContext = ctxName
	}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	restConfig, err := cc.ClientConfig()
	if err != nil {
		return nil, err
	}
	rawCfg, err := cc.RawConfig()
	if err != nil {
		return nil, err
	}
	currentCtx := rawCfg.CurrentContext
	if ctxName != "" {
		currentCtx = ctxName
	}
	core, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	disc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &Client{
		RESTConfig: restConfig,
		Core:       core,
		Dynamic:    dyn,
		Discovery:  disc,
		CurrentCtx: currentCtx,
	}, nil
}

func (c *Client) GetClusterScoped(ctx context.Context, gvr schema.GroupVersionResource, name string) (*unstructured.Unstructured, error) {
	return c.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) ListClusterScoped(ctx context.Context, gvr schema.GroupVersionResource) (*unstructured.UnstructuredList, error) {
	return c.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
}

func (c *Client) ApplyClusterScoped(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if obj.GetResourceVersion() == "" {
		return c.Dynamic.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	}
	return c.Dynamic.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
}

func (c *Client) ReadTLSSecret(ctx context.Context, namespace, name string) (certPEM, keyPEM []byte, err error) {
	secret, err := c.Core.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	return secret.Data["tls.crt"], secret.Data["tls.key"], nil
}

func (c *Client) ReadSecretValue(ctx context.Context, namespace, name, key string) ([]byte, error) {
	secret, err := c.Core.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	value, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s missing key %q", namespace, name, key)
	}
	return value, nil
}

func (c *Client) DefaultStorageClass(ctx context.Context) (string, error) {
	scs, err := c.Core.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, sc := range scs.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc.Name, nil
		}
	}
	return "", nil
}

func (c *Client) StartPortForward(ctx context.Context, namespace, service string, remotePort int) (*PortForward, error) {
	ep, err := c.Core.CoreV1().Endpoints(namespace).Get(ctx, service, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var podName string
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				podName = addr.TargetRef.Name
				break
			}
		}
		if podName != "" {
			break
		}
	}
	if podName == "" {
		return nil, fmt.Errorf("no pod endpoint found for service %s/%s", namespace, service)
	}

	localLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	localPort := localLn.Addr().(*net.TCPAddr).Port
	_ = localLn.Close()

	hostIP := strings.TrimPrefix(c.RESTConfig.Host, "https://")
	hostIP = strings.TrimPrefix(hostIP, "http://")
	serverURL := &url.URL{
		Scheme: "https",
		Path:   fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName),
		Host:   hostIP,
	}

	transport, upgrader, err := spdy.RoundTripperFor(c.RESTConfig)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, serverURL)
	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{})
	pf, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopCh, readyCh, io.Discard, io.Discard)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		select {
		case stopCh <- struct{}{}:
		default:
		}
	}()

	go func() {
		_ = pf.ForwardPorts()
	}()

	select {
	case <-readyCh:
	case <-time.After(10 * time.Second):
		select {
		case stopCh <- struct{}{}:
		default:
		}
		return nil, fmt.Errorf("timed out waiting for port-forward to become ready")
	}

	return &PortForward{LocalPort: localPort, stopCh: stopCh, readyCh: readyCh}, nil
}

func (p *PortForward) Close() {
	if p == nil {
		return
	}
	select {
	case p.stopCh <- struct{}{}:
	default:
	}
}

func DecodeB64(in string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(in)
}

func TLSCertificate(certPEM, keyPEM []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(certPEM, keyPEM)
}
