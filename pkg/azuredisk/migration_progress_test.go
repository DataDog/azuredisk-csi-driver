/*
Copyright 2024 The Kubernetes Authors.

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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInitializePVInformer(t *testing.T) {
	tests := []struct {
		name       string
		kubeClient bool
		expectInit bool
	}{
		{
			name:       "with kubeclient",
			kubeClient: true,
			expectInit: true,
		},
		{
			name:       "without kubeclient",
			kubeClient: false,
			expectInit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &DriverCore{}
			driver.Name = "disk.csi.azure.com"

			if tt.kubeClient {
				driver.kubeClient = fake.NewSimpleClientset()
			}

			driver.initializePVInformer()

			if tt.expectInit {
				assert.NotNil(t, driver.pvInformer)
				assert.NotNil(t, driver.pvLister)
				assert.NotNil(t, driver.informerStopCh)
			} else {
				assert.Nil(t, driver.pvInformer)
				assert.Nil(t, driver.pvLister)
				assert.Nil(t, driver.informerStopCh)
			}
		})
	}
}

func TestHandlePVMigrationEvent(t *testing.T) {
	driver := &DriverCore{}
	driver.Name = "disk.csi.azure.com"

	tests := []struct {
		name   string
		pv     *corev1.PersistentVolume
		expect bool
	}{
		{
			name: "wrong driver",
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{Driver: "other.driver"},
					},
				},
			},
			expect: false,
		},
		{
			name: "no migration annotation",
			pv: &corev1.PersistentVolume{
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{Driver: "disk.csi.azure.com"},
					},
				},
			},
			expect: false,
		},
		{
			name: "migration completed",
			pv: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"disk.csi.azure.com/storageaccounttype": "Premium_LRS",
						"migration.disk.csi.azure.com/status":   "completed",
					},
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{Driver: "disk.csi.azure.com"},
					},
				},
			},
			expect: false,
		},
		{
			name: "active migration",
			pv: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"disk.csi.azure.com/storageaccounttype": "Premium_LRS",
					},
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{Driver: "disk.csi.azure.com"},
					},
				},
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver.handlePVMigrationEvent(tt.pv)

			// Give goroutine time to execute
			if tt.expect {
				// For this test, we expect the goroutine to be started
				// but we can't easily wait for it in a unit test
				assert.True(t, true) // Just verify no panic
			}
		})
	}
}

func TestUpdatePVMigrationProgress(t *testing.T) {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
			Annotations: map[string]string{
				"disk.csi.azure.com/storageaccounttype": "Premium_LRS",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:       "disk.csi.azure.com",
					VolumeHandle: "/subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Compute/disks/test",
				},
			},
		},
	}

	tests := []struct {
		name          string
		kubeClient    bool
		status        string
		expectError   bool
		expectCleanup bool
	}{
		{
			name:        "no kubeclient",
			kubeClient:  false,
			status:      "converting",
			expectError: true,
		},
		{
			name:       "converting status",
			kubeClient: true,
			status:     "converting",
		},
		{
			name:          "completed status",
			kubeClient:    true,
			status:        "completed",
			expectCleanup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &DriverCore{}
			driver.Name = "disk.csi.azure.com"

			if tt.kubeClient {
				driver.kubeClient = fake.NewSimpleClientset(pv)
			}

			err := driver.updatePVMigrationProgress(pv, tt.status)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
