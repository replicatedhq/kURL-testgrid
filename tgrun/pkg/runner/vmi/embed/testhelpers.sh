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
