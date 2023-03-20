#!/bin/bash

# object store functions (create bucket, write object, get object)
function object_store_bucket_exists() {
    local bucket=$1
    local acl="x-amz-acl:private"
    local d=$(LC_TIME="en_US.UTF-8" TZ="UTC" date +"%a, %d %b %Y %T %z")
    local string="HEAD\n\n\n${d}\n${acl}\n/$bucket"
    local sig=$(echo -en "${string}" | openssl sha1 -hmac "${OBJECT_STORE_SECRET_KEY}" -binary | base64)

    curl -fsSL -I \
        --globoff \
        --noproxy "*" \
        -H "Host: $OBJECT_STORE_CLUSTER_IP" \
        -H "Date: $d" \
        -H "$acl" \
        -H "Authorization: AWS $OBJECT_STORE_ACCESS_KEY:$sig" \
        "http://$OBJECT_STORE_CLUSTER_IP/$bucket"
}

function _object_store_create_bucket() {
  local bucket=$1
  local acl="x-amz-acl:private"
  local d=$(LC_TIME="en_US.UTF-8" TZ="UTC" date +"%a, %d %b %Y %T %z")
  local string="PUT\n\n\n${d}\n${acl}\n/$bucket"
  local sig=$(echo -en "${string}" | openssl sha1 -hmac "${OBJECT_STORE_SECRET_KEY}" -binary | base64)
  curl -fsSL -X PUT  \
    --globoff \
    --noproxy "*" \
    -H "Host: $OBJECT_STORE_CLUSTER_IP" \
    -H "Date: $d" \
    -H "$acl" \
    -H "Authorization: AWS $OBJECT_STORE_ACCESS_KEY:$sig" \
    "http://$OBJECT_STORE_CLUSTER_IP/$bucket"
}

function object_store_create_bucket() {
    if object_store_bucket_exists "$1" ; then
        return 0
    fi
    if ! _object_store_create_bucket "$1" ; then
        echo "failed to create bucket $1"
        return 1
    fi
    echo "object store bucket $1 created"
}

function object_store_write_object() {
  local bucket=$1
  local file=$2
  local resource="/${bucket}/${file}"
  local contentType="application/x-compressed-tar"
  local d=$(LC_TIME="en_US.UTF-8" TZ="UTC" date +"%a, %d %b %Y %T %z")
  local string="PUT\n\n${contentType}\n${d}\n${resource}"
  local sig=$(echo -en "${string}" | openssl sha1 -hmac "${OBJECT_STORE_SECRET_KEY}" -binary | base64)

  curl -X PUT -T "${file}" \
    --globoff \
    --noproxy "*" \
    -H "Host: $OBJECT_STORE_CLUSTER_IP" \
    -H "Date: $d" \
    -H "Content-Type: ${contentType}" \
    -H "Authorization: AWS $OBJECT_STORE_ACCESS_KEY:$sig" \
    "http://$OBJECT_STORE_CLUSTER_IP$resource"
}

function object_store_get_object() {
  local bucket=$1
  local file=$2
  local resource="/${bucket}/${file}"
  local contentType="application/x-compressed-tar"
  local d=$(LC_TIME="en_US.UTF-8" TZ="UTC" date +"%a, %d %b %Y %T %z")
  local string="GET\n\n${contentType}\n${d}\n${resource}"
  local sig=$(echo -en "${string}" | openssl sha1 -hmac "${OBJECT_STORE_SECRET_KEY}" -binary | base64)

  curl -X GET -o "${file}" \
  --globoff \
  --noproxy "*" \
  -H "Host: $OBJECT_STORE_CLUSTER_IP" \
  -H "Date: $d" \
  -H "Content-Type: ${contentType}" \
  -H "Authorization: AWS $OBJECT_STORE_ACCESS_KEY:$sig" \
  "http://$OBJECT_STORE_CLUSTER_IP$resource"
}

# dump_longhorn_logs prints the logs of all pods in the longhorn-system namespace.
function dump_longhorn_logs() {
    for pod in $(kubectl get pods --no-headers -n longhorn-system -o custom-columns=:.metadata.name); do
        echo "logs for longhorn-system/$pod"
        kubectl logs -n longhorn-system "$pod"
    done
}

function rook_ceph_object_store_info() {
    export OBJECT_STORE_ACCESS_KEY=$(kubectl -n rook-ceph get secret rook-ceph-object-user-rook-ceph-store-kurl -o yaml | grep AccessKey | head -1 | awk '{print $2}' | base64 --decode)
    export OBJECT_STORE_SECRET_KEY=$(kubectl -n rook-ceph get secret rook-ceph-object-user-rook-ceph-store-kurl -o yaml | grep SecretKey | head -1 | awk '{print $2}' | base64 --decode)
    export OBJECT_STORE_CLUSTER_IP=$(kubectl -n rook-ceph get service rook-ceph-rgw-rook-ceph-store | tail -n1 | awk '{ print $3}')
}

# TODO: remove rook_ecph_object_store_info once we have removed from github.com/replicatedhq/kURL
function rook_ecph_object_store_info() {
    rook_ceph_object_store_info
}

function minio_object_store_info() {
    export OBJECT_STORE_ACCESS_KEY=$(kubectl -n minio get secret minio-credentials -ojsonpath='{ .data.MINIO_ACCESS_KEY }' | base64 --decode)
    export OBJECT_STORE_SECRET_KEY=$(kubectl -n minio get secret minio-credentials -ojsonpath='{ .data.MINIO_SECRET_KEY }' | base64 --decode)
    export OBJECT_STORE_CLUSTER_IP=$(kubectl -n minio get service minio | tail -n1 | awk '{ print $3}')
}

# creates a file named after the second parameter and uploads it to the bucket in the first parameter
function make_testfile() {
    local bucket=$1
    local file=$2

    echo "writing ${file} to ${bucket} bucket"
    echo "Hello, World!" > "${file}"
    date >> "${file}"
    echo "${file} contents:"
    cat "${file}"
    object_store_write_object "${bucket}" "${file}"
}

# given a bucket that a file is stored in, and the local name of the file, gets the file from the bucket and compares it to the local copy
function validate_testfile() {
    local bucket=$1
    local file=$2

    echo "retrieving ${file} from ${bucket} bucket"
    mv "${file}" "${file}.bak"
    object_store_get_object "${bucket}" "${file}"

    echo "comparing retrieved ${file} with local copy"
    if diff "${file}" "${file}.bak"; then
        echo "${file} was successfully stored and retrieved"
        rm "${file}.bak"
    else
        echo "${file} contents:"
        cat "${file}"
        echo "${file}.bak contents:"
        cat "${file}.bak"
        return 1
    fi
}

# create the provided bucket if it does not yet exist, write a file to it, and read that file back
function validate_read_write_object_store() {
    local bucket=$1
    local file=$2

    object_store_create_bucket "$bucket"

    make_testfile "$bucket" "$file"

    validate_testfile "$bucket" "$file"
}

# wait_for_minio_ready waits up to 20s for the minio pod to be running and ready
function wait_for_minio_ready() {
    local minio_phase=
    for i in {1..5}; do
      minio_phase="$(kubectl -n minio get pods -l app=minio -o jsonpath='{.items[*].status.phase}')"
      if [ "$minio_phase" = "Running" ]; then
        local minio_cluster_ip=
        minio_cluster_ip="$(kubectl -n minio get svc minio -o jsonpath='{.spec.clusterIP}')"
        if curl -f "http://$minio_cluster_ip/minio/health/live" ; then
          if curl -f "http://$minio_cluster_ip/minio/health/ready" ; then
            break
          fi
        fi
      fi
      if [ "$i" = "5" ]; then
        echo "Minio not ready"
        kubectl -n minio get pods
        return 1
      fi
      sleep 5
    done
}

# create_deployment_with_mounted_volume creates a deployment with one replica, this deployment mounts a PVC
# (using the default storage class) on provided $mountpoint argument. Returns as soon as `kubectl rollout`
# says the deployment has been rolled out. Requires 3 arguments: the deployment name and namespace, and the
# mount point, the 4th argument is optional and is a path to the container image to be used by the deploy
# (if not provided then "nginx" is used). The PVC is named after the deployment name and the deployment uses
# `app=$deployment` as selection labels.
function create_deployment_with_mounted_volume() {
    local deployment=$1
    local namespace=$2
    local mountpoint=$3
    local image=${4:-nginx}

    echo "creating pvc $deployment in $namespace namespace"
    kubectl create -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "$deployment"
  namespace: "$namespace"
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF

    echo "creating deployment $deployment in $namespace namespace (label app=$deployment, image=$image)"
    kubectl create -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "$deployment"
  namespace: "$namespace"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: "$deployment"
  template:
    metadata:
      labels:
        app: "$deployment"
    spec:
      volumes:
        - name: pvc
          persistentVolumeClaim:
            claimName: "$deployment"
      containers:
        - name: container
          image: "$image"
          volumeMounts:
            - mountPath: "$mountpoint"
              name: pvc
EOF

    echo "waiting deployment $deployment in $namespace roll out"
    kubectl rollout status deployment "$deployment" -n "$namespace" --timeout=120s
}

# create_random_file_and_upload_to_deployment generates a 1MB random content file and uploads it to all Pods
# containing `app=$deployment` labels. Receives as argument the deployment name, the namespace, a temporary
# path (in the local host filesystem) where the random file is going to be stored. file is stored into the
# $destination path inside the Pod.
function create_random_file_and_upload_to_deployment() {
    local deployment=$1
    local namespace=$2
    local tmp_file_path=$3
    local destination=$4
    echo "generating random file under $tmp_file_path"
    dd if=/dev/urandom of="$tmp_file_path" bs=1MB count=1
    for pod in $(kubectl get --no-headers pods -n "$namespace" -l "app=$deployment" -o custom-columns=:.metadata.name); do
        echo "copying $tmp_file_path file to $pod:$destination"
        kubectl cp -n "$namespace" "$tmp_file_path" "$pod:$destination"
    done
}

# download_file_from_deployment_and_compare copies a file from Pods containing `app=$deployment` labels
# and compares it with a local copy ($tmp_file_path argument). receives as argument the deployment name
# (used to assemble the `app=$deployment` label), the namespace, the local file and the remote (inside
# the Pod) file paths.
function download_file_from_deployment_and_compare() {
    local deployment=$1
    local namespace=$2
    local tmp_file_path=$3
    local remote_file_path=$4
    local local_sha=
    local_sha=$(sha256sum < "$tmp_file_path")

    echo "comparing local file $tmp_file_path remotely"
    for pod in $(kubectl get --no-headers pods -n "$namespace" -l "app=$deployment" -o custom-columns=:.metadata.name); do
        echo "comparing local file $tmp_file_path with remote $pod:$remote_file_path"
        local remote_sha=
        remote_sha=$(kubectl exec -n "$namespace" "$pod" -- cat "$remote_file_path" | sha256sum)
        if [ "$local_sha" != "$remote_sha" ]; then
            echo "File content mismatch, expected sha $local_sha, found $remote_sha"
            exit 1
        fi
    done
}

# pvc_uses_provisioner checks if provided pvc uses the provided storage class provisioner. Gets first
# the storage class in use and then grep the storage class for the provided provisioner string.
function pvc_uses_provisioner() {
    local pvc=$1
    local namespace=$2
    local provisioner=$3
    echo "verifying if pvc $pvc uses provisioner $provisioner"
    local sc=
    sc=$(kubectl get pvc -n "$namespace" "$pvc" --no-headers -o custom-columns=:.spec.storageClassName)
    if ! kubectl get sc "$sc" --no-headers | grep -q "$provisioner"; then
        echo "pvc $pvc does not use provisioner $provisioner"
        exit 1
    fi
    echo "pvc $pvc uses provisioner $provisioner"
}

# test_push_image_to_registry spawns a job to copy an image from docker.io into the local registry.
# The job is created in the default namespace and it is deleted after the image is copied. This
# function has a timeout of 5 minutes. Image is pushed to <internal_docker_addr>/test/test:latest.
function test_push_image_to_registry() {
    echo "Trying to copy an image to kURL internal registry"
    local registry_addr
    registry_addr=$(kubectl get svc -n kurl registry -o jsonpath="{.spec.clusterIP}" 2>/dev/null)
    if [ -z "$registry_addr" ]; then
        echo "Failed to get registry address"
        exit 1
    fi

    kubectl create -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: skopeo-copy
  namespace: default
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      volumes:
      - name: authfile
        secret:
          secretName: registry-creds
      containers:
      - name: skopeo
        image: quay.io/skopeo/stable:latest
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: authfile
          mountPath: /auth
        args:
        - copy
        - --dest-tls-verify=false
        - --dest-authfile=/auth/.dockerconfigjson
        - docker://docker.io/library/registry:2.8.1
        - docker://${registry_addr}/test/test:latest
EOF

    echo "Job created, waiting for completion"
    if ! kubectl wait --for=condition=complete job/skopeo-copy -n default --timeout=5m; then
        echo "Job failed"
        kubectl get job/skopeo-copy -n default -o yaml
        kubectl logs -n default job/skopeo-copy
        kubectl delete job/skopeo-copy -n default
        echo "Failed to copy image to registry"
        exit 1
    fi

    echo "Job finished"
    kubectl logs -n default job/skopeo-copy
    kubectl delete job/skopeo-copy -n default
}

# test_pull_image_from_registry spawns a job that pulls an image from the local registry and stores
# it in a local tar file inside the container. The job is created in the default namespace and it is
# deleted after the image is copied. This function has a timeout of 5 minutes. Image is pulled from
# <internal_docker_addr>/test/test:latest.
function test_pull_image_from_registry() {
    echo "Trying to pull an image from kURL internal registry"
    local registry_addr
    registry_addr=$(kubectl get svc -n kurl registry -o jsonpath="{.spec.clusterIP}" 2>/dev/null)
    if [ -z "$registry_addr" ]; then
        echo "Failed to get registry address"
        exit 1
    fi

    kubectl create -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: skopeo-pull
  namespace: default
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      volumes:
      - name: authfile
        secret:
          secretName: registry-creds
      containers:
      - name: skopeo
        image: quay.io/skopeo/stable:latest
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: authfile
          mountPath: /auth
        args:
        - copy
        - --src-tls-verify=false
        - --src-authfile=/auth/.dockerconfigjson
        - docker://${registry_addr}/test/test:latest
        - docker-archive:/image.tar
EOF

    echo "Job created, waiting for completion"
    if ! kubectl wait --for=condition=complete job/skopeo-pull -n default --timeout=5m; then
        kubectl logs -n default job/skopeo-pull
        kubectl delete job/skopeo-pull -n default
        echo "Failed to copy image to registry"
        exit 1
    fi

    echo "Job finished"
    kubectl logs -n default job/skopeo-pull
    kubectl delete job/skopeo-pull -n default
}

# install_and_customize_kurl_integration_test_application deploys an application created solely for
# integration test purposes. this application is a composed by a simple nginx serving a html page,
# this page has a simple string as its content, the content of the string is defined by means of
# a configuration called 'variable', i.e. whatever is set in 'variable' is printed as the output.
# this function installs the application on the cluster and customize 'variable', then awaits for
# the deployment to take place by attempting to reach the application through curl. this function
# requires one argument: a path for a license file.
function install_and_customize_kurl_integration_test_application() {
  echo "attempting to install kurl integration test application."
  local license_path=$1
  if ! kubectl kots install --license-file=$license_path --namespace=default kurl-integration-test-application/stable ; then
    echo "failed to install kurl integration test application."
    exit 1
  fi
  echo "kurl integration test application has been successfully installed."

  echo "attempting to customize kurl integration test application."
  if ! kubectl kots set config kurl-integration-test-application --key variable --value installation --deploy ; then
    echo "failed to customize kurl integration test application."
    exit 1
  fi
  echo "kurl integration test application customized, waiting 2m for the deployment."

  local svc_ip
  local app_content
  for i in $(seq 1 24); do
    svc_ip=$(kubectl -n default get service nginx | tail -n1 | awk '{ print $3}')
    app_content=$(curl -s $svc_ip 2>&1)
    if [ "$app_content" == "installation" ]; then
      break
    fi

    echo "attempt $i to read kurl integration test application ($svc_ip) failed, result:"
    echo $app_content
    sleep 5
  done

  if [ "$app_content" != "installation" ]; then
    svc_ip=$(kubectl -n default get service nginx | tail -n1 | awk '{ print $3}')
    echo "error reading kurl integration test application ($svc_ip), result:"
    curl -vvv $svc_ip
    exit 1
  fi

  echo "kurl integration test application deployed successfully."
}

# check_and_customize_kurl_integration_test_application verifies if the kurl integration test app
# is running with the config applied by install_and_customize_kurl_integration_test_application
# then attempts to change the application config and redeploy it. awaits for the deployment after
# the customization to roll out.
function check_and_customize_kurl_integration_test_application() {
  local svc_ip
  local app_content
  echo "checking if kurl integration test application is still running with the previously customized configuration."
  for i in $(seq 1 12); do
    svc_ip=$(kubectl -n default get service nginx | tail -n1 | awk '{ print $3}')
    app_content=$(curl -s $svc_ip 2>&1)
    if [ "$app_content" == "installation" ]; then
      break
    fi

    echo "attempt $i to reach kurl integration test application ($svc_ip) failed, result:"
    echo $app_content
    sleep 5
  done

  if [ "$app_content" != "installation" ]; then
    svc_ip=$(kubectl -n default get service nginx | tail -n1 | awk '{ print $3}')
    echo "error acessing kurl integration application, unexpected result when curling app:"
    curl -vvv $svc_ip
    exit 1
  fi
  echo "kurl integration test application is running with the previosly customized config."

  echo "updating kurl integration test application configuration."
  if ! kubectl kots set config kurl-integration-test-application --key variable --value upgrade --deploy ; then
    echo "failed to update kurl integration test application configuration."
    exit 1
  fi
  echo "kurl integration test application customized successfully, awaiting for the deployment."

  for i in $(seq 1 24); do
    svc_ip=$(kubectl -n default get service nginx | tail -n1 | awk '{ print $3}')
    app_content=$(curl -s $svc_ip 2>&1)
    if [ "$app_content" == "upgrade" ]; then
      break
    fi

    echo "attempt $i to reach kurl integration test application ($svc_ip) failed, result:"
    echo $app_content
    sleep 5
  done

  if [ "$app_content" != "upgrade" ]; then
    echo "error acessing kurl integration test application, unexpected result when curling app:"
    curl -vvv $svc_ip
    exit 1
  fi

  echo "kurl integration test application still working after the upgrade."
}
