package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
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
			name: "sendlogs succeeded 31m is not reaped by this path",
			vmi:  vmiWithAge("abc123-sendlogs", kubevirtv1.Succeeded, 31*time.Minute),
			want: false,
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
