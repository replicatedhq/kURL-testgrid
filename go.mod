module github.com/replicatedhq/kurl-testgrid

go 1.19

require (
	filippo.io/age v1.1.1
	github.com/DataDog/datadog-go v4.8.3+incompatible
	github.com/aws/aws-sdk-go v1.44.224
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.7
	github.com/pkg/errors v0.9.1
	github.com/replicatedhq/kurlkinds v1.1.0
	github.com/replicatedhq/troubleshoot v0.57.1
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.8.2
	go.uber.org/zap v1.24.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.26.1
	k8s.io/apimachinery v0.26.1
	k8s.io/client-go v0.26.1
	kubevirt.io/api v0.59.0
	kubevirt.io/client-go v0.58.1
)

require (
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/coreos/prometheus-operator v0.38.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/mock v1.5.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v0.0.0-20191119172530-79f836b90111 // indirect
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/openshift/api v0.0.0-20211217221424-8779abfbd571 // indirect
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47 // indirect
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/afero v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/oauth2 v0.5.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.4.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.26.1 // indirect
	k8s.io/klog/v2 v2.90.0 // indirect
	k8s.io/kube-openapi v0.0.0-20230202010329-39b3636cbaa3 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	kubevirt.io/containerized-data-importer-api v1.55.0 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk/api v0.0.0-20220329064328-f3cc58c6ed90 // indirect
	sigs.k8s.io/controller-runtime v0.14.4 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	// from github.com/replicatedhq/troubleshoot and github.com/kubevirt/client-go
	github.com/go-ole/go-ole => github.com/go-ole/go-ole v1.2.6 // needed for arm builds
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210105115604-44119421ec6b
	k8s.io/api => k8s.io/api v0.23.10
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.10
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.10
	k8s.io/apiserver => k8s.io/apiserver v0.23.10
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.10
	k8s.io/client-go => k8s.io/client-go v0.23.10
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.10
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.10
	k8s.io/code-generator => k8s.io/code-generator v0.23.10
	k8s.io/component-base => k8s.io/component-base v0.23.10
	k8s.io/cri-api => k8s.io/cri-api v0.23.10
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.10
	k8s.io/klog => k8s.io/klog v0.4.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.10
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.10
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.10
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.10
	k8s.io/kubectl => k8s.io/kubectl v0.23.10
	k8s.io/kubelet => k8s.io/kubelet v0.23.10
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.10
	k8s.io/metrics => k8s.io/metrics v0.23.10
	k8s.io/node-api => k8s.io/node-api v0.23.10
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.10
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.10
	k8s.io/sample-controller => k8s.io/sample-controller v0.23.10
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.11.2 // k8s 1.23
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.8.0 // k8s 1.23
)

exclude (
	k8s.io/client-go v1.4.0
	k8s.io/client-go v1.5.0
	k8s.io/client-go v1.5.1
	k8s.io/client-go v1.5.2
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/client-go v2.0.0+incompatible
	k8s.io/client-go v2.0.0-alpha.1+incompatible
	k8s.io/client-go v3.0.0+incompatible
	k8s.io/client-go v3.0.0-beta.0+incompatible
	k8s.io/client-go v4.0.0+incompatible
	k8s.io/client-go v4.0.0-beta.0+incompatible
	k8s.io/client-go v5.0.0+incompatible
	k8s.io/client-go v5.0.1+incompatible
	k8s.io/client-go v6.0.0+incompatible
	k8s.io/client-go v7.0.0+incompatible
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/client-go v9.0.0+incompatible
	k8s.io/client-go v9.0.0-invalid+incompatible
)
