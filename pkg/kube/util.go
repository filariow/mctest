/*
Copyright © 2023 Francesco Ilario

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package kube

import (
	"fmt"
	"os"
	"path"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var ErrInvalidAuthInfo error = fmt.Errorf("error invalid authentication info in rest.Config")

func GetCurrentContextClient() (*kubernetes.Clientset, error) {
	cfg, err := GetRESTConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func GetConfigDefaultNamespace() (*string, error) {
	cc, err := getClientConfig()
	if err != nil {
		return nil, err
	}
	ns, _, err := cc.Namespace()
	if err != nil {
		return nil, err
	}

	return &ns, err
}

func GetRESTConfig() (*rest.Config, error) {
	cc, err := getClientConfig()
	if err != nil {
		return nil, err
	}

	cfg, err := cc.ClientConfig()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getClientConfig() (clientcmd.ClientConfig, error) {
	kp, err := getKubeconfigPath()
	if err != nil {
		return nil, err
	}

	kd, err := os.ReadFile(*kp)
	if err != nil {
		return nil, err
	}

	return clientcmd.NewClientConfigFromBytes(kd)
}

func getKubeconfigPath() (*string, error) {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return &env, nil
	}

	hd, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	kp := path.Join(hd, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName)
	return &kp, nil
}

func BuildClient(kfg []byte) (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kfg)
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func SerializeKubeconfig(cfg *rest.Config, namespace *string) ([]byte, error) {
	ai, err := buildAuthInfo(cfg)
	if err != nil {
		return nil, err
	}

	cl := map[string]*clientcmdapi.Cluster{
		"default-cluster": {
			Server:                   cfg.Host,
			CertificateAuthorityData: cfg.CAData,
		},
	}

	ct := map[string]*clientcmdapi.Context{
		"default-context": {
			Cluster:   "default-cluster",
			AuthInfo:  "default",
			Namespace: "default",
		},
	}
	if namespace != nil {
		ct["default-context"].Namespace = *namespace
	}

	cc := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       cl,
		Contexts:       ct,
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{"default": ai},
		CurrentContext: "default-context",
	}

	return clientcmd.Write(cc)
}

func buildAuthInfo(cfg *rest.Config) (*clientcmdapi.AuthInfo, error) {
	if cfg.BearerToken != "" {
		return &clientcmdapi.AuthInfo{Token: cfg.BearerToken}, nil
	}

	if cfg.CertData != nil && cfg.KeyData != nil {
		return &clientcmdapi.AuthInfo{
			ClientCertificateData: cfg.CertData,
			ClientKeyData:         cfg.KeyData,
		}, nil
	}

	return nil, fmt.Errorf("%w: neither bearer token nor CertData/KeyData found", ErrInvalidAuthInfo)
}
