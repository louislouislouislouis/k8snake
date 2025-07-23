/*
Package services provides services for interacting with external systems.
It contains functionality for Kubernetes operations, and Keycloak token management.
*/
package services

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/louislouislouislouis/k8snake/pkg/utils"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/util/homedir"
)

type K8sService struct {
	config *rest.Config
	client *kubernetes.Clientset
}

func NewK8sService() (*K8sService, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	return &K8sService{
		config: config,
		client: clientset,
	}, nil
}

type PFCommunication struct {
	stopChan  chan struct{}
	startChan chan struct{}
	LocalPort int
}

func (pfKomm *PFCommunication) Close() {
	pfKomm.stopChan <- struct{}{}
}

func (pfKomm *PFCommunication) IsStarted() bool {
	return pfKomm.startChan != nil
}

func (k8s *K8sService) GetSecret(namespace string, name string, ctx context.Context) (*v1.Secret, error) {
	secret, err := k8s.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %v", err)
	}
	if secret == nil {
		return nil, fmt.Errorf("secret %s not found in namespace %s", name, namespace)
	}
	return secret, nil
}

func (k8s *K8sService) PortForwardSVC(namespace string, podName string, port int) (*PFCommunication, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(k8s.config)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimLeft(k8s.config.Host, "htps:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	localPort, err := utils.GetFreePort()
	if err != nil {
		return nil, err
	}

	portStr := fmt.Sprintf("%d:%d", localPort, port)

	forwarder, err := portforward.New(dialer, []string{portStr}, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}

	go func() error {
		if err = forwarder.ForwardPorts(); err != nil { // Locks until stopChan is closed.
			return fmt.Errorf("error forwarding ports: %v", err)
		}
		log.Debug().Msg("Port forwarding stopped")
		return nil
	}()
	select {
	case <-readyChan:
		log.Debug().Msgf("Port forwarding is ready on local port %d -> %d", localPort, port)
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for port forwarding to be ready")
	}

	if len(errOut.String()) != 0 {
		return nil, fmt.Errorf("error found during init port-forward : %v", errOut.String())
	} else if len(out.String()) != 0 {
		log.Debug().Msgf("Found in forward output channel : %s", out.String())
	}

	log.Debug().Msgf("Port forwarding started on local port %d to remote port %d", localPort, port)

	return &PFCommunication{
		stopChan:  stopChan,
		startChan: readyChan,
		LocalPort: localPort,
	}, nil
}

func (k8s *K8sService) GetPodsForSvc(svc *v1.Service, namespace string, ctx context.Context) (*v1.PodList, error) {
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := k8s.client.CoreV1().Pods(namespace).List(ctx, listOptions)
	return pods, err
}

func (k8s *K8sService) GetService(namespace string, name string, ctx context.Context) (*v1.Service, error) {
	svc, err := k8s.client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("error getting svc: %v", err)
		return nil, err
	}
	if svc == nil {
		return nil, fmt.Errorf("service %s not found in namespace %s", name, namespace)
	}
	return svc, nil
}
