# Testgrid

Testgrid is a an automation testing platform for kURL.
Testgrid installs kURL specs on a set of linux os versions and runs conformance tests.

## Testgrid Results

Testgrid results can be viewed at https://testgrid.kurl.sh/

### Debugging failed runs

For each failed Testgrid run, a [Support Bundle](https://github.com/replicatedhq/troubleshoot) is collected, encrypted and uploaded to a public S3 bucket.

The bundle URL can be obtained from the Testgrid logs output.

The bundle is encrypted with a passphrase using the [age](https://github.com/FiloSottile/age) file encryption tool.

## Run Testgrid locally

### Prerequisites

- Have docker running locally
- Have some k8s cluster running
- Install [SchemaHero](https://schemahero.io/docs/installing/kubectl/)
- Install skaffold: https://skaffold.dev/docs/install/
- Set `GOOS` and `GOARCH`
```bash
   export GOOS=linux
   export GOARCH=amd64
```

### Run Testgrid using Skaffold

- Connect to the cluster
- Run the following command from ``TESTGRID`` path
```
make install
```

- Setup port-forwards
``` bash
kubectl port-forward svc/tgapi 30110:3000 &
kubectl port-forward svc/testgrid-web 30881:8080
```

- Now you are ready to do your first test. 

- From tgrun folder run the following command
```
./bin/tgrun queue --os-spec hack/os-spec.yaml --spec hack/test-spec.yaml --ref test-1 --api-token this-is-super-secret --api-endpoint http://localhost:30110
```

- From the web service you should be able to see the pending test.

- Now time to setup your runner by using ``terraform`` go indide the ``deploy` folder and follow the steps from the readme file.

### Run Testgrid on Okteto

1. Change directories to the root of the project
1. Run `okteto pipeline deploy`
1. To "queue" a run `./bin/tgrun queue --os-spec hack/os-spec.yaml --spec hack/test-spec.yaml --ref ethan-1 --api-token this-is-super-secret --api-endpoint https://tgapi-${OKTETO_NAMESPACE}.okteto.repldev.com`
