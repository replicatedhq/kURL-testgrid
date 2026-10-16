package runner

import (
	"errors"
	"io"
	"os"
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

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was printed to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

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

// Test_cleanUpVMIs_doubleDeleteSendlogs covers the reap-switch branches of
// the sendlogs reap-on-a-short-leash path at the cleanUpVMIs integration
// level: a Succeeded VMI already removed by the succeeded-cleanup path in
// the same pass (double-delete NotFound, must not be logged as a failure or
// as a reap), a Running VMI stuck past its leash (single successful
// delete), and a delete that fails with a real, non-NotFound error (logged,
// not propagated).
func Test_cleanUpVMIs_doubleDeleteSendlogs(t *testing.T) {
	tests := []struct {
		name          string
		vmi           kubevirtv1.VirtualMachineInstance
		deleteResults []error
		wantReapLog   bool
		wantReason    string
	}{
		{
			name:          "succeeded sendlogs already deleted earlier in this pass: second delete NotFound, no reap log",
			vmi:           vmiWithAge("abc123-initialprimary-sendlogs", kubevirtv1.Succeeded, 91*time.Minute),
			deleteResults: []error{nil, apierrors.NewNotFound(schema.GroupResource{Resource: "virtualmachineinstances"}, "abc123-initialprimary-sendlogs")},
			wantReapLog:   false,
		},
		{
			name:          "running sendlogs stuck past its leash: single delete, reaped with the running reason",
			vmi:           vmiWithAge("def456-sendlogs", kubevirtv1.Running, 31*time.Minute),
			deleteResults: []error{nil},
			wantReapLog:   true,
			wantReason:    "without finishing log shipping",
		},
		{
			name:          "succeeded sendlogs stuck past its leash but not yet caught by the succeeded-cleanup window: single delete, reaped with the succeeded reason",
			vmi:           vmiWithAge("jkl012-sendlogs", kubevirtv1.Succeeded, 31*time.Minute),
			deleteResults: []error{nil},
			wantReapLog:   true,
			wantReason:    "after finishing log shipping",
		},
		{
			name:          "sendlogs delete fails with a non-NotFound error: logged as a failure, not a reap, and not propagated",
			vmi:           vmiWithAge("ghi789-sendlogs", kubevirtv1.Running, 31*time.Minute),
			deleteResults: []error{errors.New("etcd unavailable")},
			wantReapLog:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			ctrl := gomock.NewController(t)

			mockVMIClient := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			mockVMIClient.EXPECT().
				List(gomock.Any(), gomock.Any()).
				Return(&kubevirtv1.VirtualMachineInstanceList{Items: []kubevirtv1.VirtualMachineInstance{tt.vmi}}, nil)
			for _, result := range tt.deleteResults {
				mockVMIClient.EXPECT().
					Delete(gomock.Any(), tt.vmi.Name, gomock.Any()).
					Return(result)
			}

			mockClient := kubecli.NewMockKubevirtClient(ctrl)
			mockClient.EXPECT().VirtualMachineInstance(Namespace).Return(mockVMIClient).AnyTimes()

			var err error
			out := captureStdout(t, func() {
				err = cleanUpVMIs(mockClient)
			})
			req.NoError(err)

			reapLog := "Reaping stuck sendlogs vmi " + tt.vmi.Name
			if tt.wantReapLog {
				req.Contains(out, reapLog)
				req.Contains(out, tt.wantReason)
			} else {
				req.NotContains(out, reapLog)
			}
		})
	}
}
