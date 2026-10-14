package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func vmiWithAge(name string, phase kubevirtv1.VirtualMachineInstancePhase, age time.Duration) kubevirtv1.VirtualMachineInstance {
	return kubevirtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-age)),
		},
		Status: kubevirtv1.VirtualMachineInstanceStatus{
			Phase: phase,
		},
	}
}

func Test_shouldReapStuckSendlogs(t *testing.T) {
	tests := []struct {
		name string
		vmi  kubevirtv1.VirtualMachineInstance
		want bool
	}{
		{
			name: "sendlogs running 31m is reaped",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Running, 31*time.Minute),
			want: true,
		},
		{
			name: "normal vmi running 31m is not reaped",
			vmi:  vmiWithAge("abc123-primary", kubevirtv1.Running, 31*time.Minute),
			want: false,
		},
		{
			name: "sendlogs running 29m is not yet reaped",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Running, 29*time.Minute),
			want: false,
		},
		{
			// exactly 30m is not a stable boundary to assert on: creation time is
			// captured relative to time.Now() and the elapsed-time check runs a
			// moment later, so "age == 30m" always reads back as slightly over.
			// Assert just inside and just outside the threshold instead.
			name: "sendlogs running 29m59s is not yet reaped",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Running, 30*time.Minute-time.Second),
			want: false,
		},
		{
			name: "sendlogs running 30m and 1s is reaped",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Running, 30*time.Minute+time.Second),
			want: true,
		},
		{
			name: "sendlogs succeeded 31m is reaped regardless of phase",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Succeeded, 31*time.Minute),
			want: true,
		},
		{
			name: "sendlogs scheduling 31m is reaped regardless of phase",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Scheduling, 31*time.Minute),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(tt.want, shouldReapStuckSendlogs(tt.vmi))
		})
	}
}

func Test_cleanupIsTimeout(t *testing.T) {
	tests := []struct {
		name string
		vmi  kubevirtv1.VirtualMachineInstance
		want bool
	}{
		{
			name: "non-running non-succeeded 91m is reaped",
			vmi:  vmiWithAge("abc123-primary", kubevirtv1.Scheduling, 91*time.Minute),
			want: true,
		},
		{
			name: "non-running non-succeeded 89m is not yet reaped",
			vmi:  vmiWithAge("abc123-primary", kubevirtv1.Scheduling, 89*time.Minute),
			want: false,
		},
		{
			name: "running 481m is reaped",
			vmi:  vmiWithAge("abc123-primary", kubevirtv1.Running, 481*time.Minute),
			want: true,
		},
		{
			name: "running 89m is not reaped",
			vmi:  vmiWithAge("abc123-primary", kubevirtv1.Running, 89*time.Minute),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(tt.want, cleanupIsTimeout(tt.vmi))
		})
	}
}

// Test_cleanUpVMIs_doubleDeleteSendlogs covers a Succeeded "-sendlogs" VMI
// old enough to be caught by both the succeeded-cleanup path and the
// sendlogs reap-on-a-short-leash path in the same pass: the first delete
// succeeds, the second returns NotFound and must not be logged as a failure.
func Test_cleanUpVMIs_doubleDeleteSendlogs(t *testing.T) {
	req := require.New(t)
	ctrl := gomock.NewController(t)

	vmi := vmiWithAge("abc123-initialprimary-sendlogs", kubevirtv1.Succeeded, 91*time.Minute)

	mockVMIClient := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	mockVMIClient.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&kubevirtv1.VirtualMachineInstanceList{Items: []kubevirtv1.VirtualMachineInstance{vmi}}, nil)
	mockVMIClient.EXPECT().
		Delete(gomock.Any(), vmi.Name, gomock.Any()).
		Return(nil)
	mockVMIClient.EXPECT().
		Delete(gomock.Any(), vmi.Name, gomock.Any()).
		Return(apierrors.NewNotFound(schema.GroupResource{Resource: "virtualmachineinstances"}, vmi.Name))

	mockClient := kubecli.NewMockKubevirtClient(ctrl)
	mockClient.EXPECT().VirtualMachineInstance(Namespace).Return(mockVMIClient).AnyTimes()

	err := cleanUpVMIs(mockClient)
	req.NoError(err)
}
