/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azuredisk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	clientSet "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"sigs.k8s.io/azuredisk-csi-driver/pkg/util"
	azure "sigs.k8s.io/cloud-provider-azure/pkg/provider"
)

var (
	// DefaultAzureCredentialFileEnv is the default azure credentials file env variable
	DefaultAzureCredentialFileEnv = "AZURE_CREDENTIAL_FILE"
	// DefaultCredFilePathLinux is default creds file for linux machine
	DefaultCredFilePathLinux = "/etc/kubernetes/azure.json"
	// DefaultCredFilePathWindows is default creds file for windows machine
	DefaultCredFilePathWindows = "C:\\k\\azure.json"
)

// IsAzureStackCloud decides whether the driver is running on Azure Stack Cloud.
func IsAzureStackCloud(cloud string, disableAzureStackCloud bool) bool {
	return !disableAzureStackCloud && strings.EqualFold(cloud, azureStackCloud)
}

// GetCloudProvider get Azure Cloud Provider
func GetCloudProvider(kubeconfig, secretName, secretNamespace, userAgent string) (*azure.Cloud, error) {
	kubeClient, err := getKubeClient(kubeconfig)
	if err != nil {
		klog.Warningf("get kubeconfig(%s) failed with error: %v", kubeconfig, err)
		if !os.IsNotExist(err) && err != rest.ErrNotInCluster {
			return nil, fmt.Errorf("failed to get KubeClient: %v", err)
		}
	}

	az := &azure.Cloud{
		InitSecretConfig: azure.InitSecretConfig{
			SecretName:      secretName,
			SecretNamespace: secretNamespace,
			CloudConfigKey:  "cloud-config",
		},
	}
	if kubeClient != nil {
		klog.V(2).Infof("reading cloud config from secret")
		az.KubeClient = kubeClient
		if err := az.InitializeCloudFromSecret(); err != nil {
			klog.V(2).Infof("InitializeCloudFromSecret failed with error: %v", err)
		}
	}

	if az.TenantID == "" || az.SubscriptionID == "" || az.ResourceGroup == "" {
		klog.V(2).Infof("could not read cloud config from secret")
		credFile, ok := os.LookupEnv(DefaultAzureCredentialFileEnv)
		if ok && strings.TrimSpace(credFile) != "" {
			klog.V(2).Infof("%s env var set as %v", DefaultAzureCredentialFileEnv, credFile)
		} else {
			if util.IsWindowsOS() {
				credFile = DefaultCredFilePathWindows
			} else {
				credFile = DefaultCredFilePathLinux
			}

			klog.V(2).Infof("use default %s env var: %v", DefaultAzureCredentialFileEnv, credFile)
		}

		config, err := os.Open(credFile)
		if err != nil {
			klog.Errorf("load azure config from file(%s) failed with %v", credFile, err)
			return nil, fmt.Errorf("load azure config from file(%s) failed with %v", credFile, err)
		}
		defer config.Close()

		klog.V(2).Infof("read cloud config from file: %s successfully", credFile)
		if az, err = azure.NewCloudWithoutFeatureGates(config, false); err != nil {
			return az, err
		}
	}

	// reassign kubeClient
	if kubeClient != nil && az.KubeClient == nil {
		az.KubeClient = kubeClient
	}
	return az, nil
}

// GetKubeConfig gets config object from config file
func GetKubeConfig(kubeconfig string) (config *rest.Config, err error) {

	if kubeconfig == "" {
		// if kubeconfig path is not passed
		// read the incluster config
		config, err = rest.InClusterConfig()

		// if we couldn't get the in-cluster config
		// get kubeconfig path from environment variable
		if err != nil {
			kubeconfig = os.Getenv("KUBECONFIG")
			if kubeconfig == "" {
				kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
			}
		} else {
			return config, err
		}
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)

	return config, err
}

func getKubeClient(kubeconfig string) (*clientSet.Clientset, error) {
	config, err := GetKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return clientSet.NewForConfig(config)
}
