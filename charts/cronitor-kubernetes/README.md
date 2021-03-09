
## Description

_To be written_

## Instructions
A valid Cronitor API key is required. Before deploying the agent, create a 
Kubernetes Secret in the namespace in which you plan to deploy this Helm chart, and 
then put the name of the Secret and the key at which the API key be found in
the following chart values: 
* `credentials.secretName`
* `credentials.secretKey`

This can be created easily using `kubectl`. As an example:

```bash
kubectl create secret generic cronitor-secret --from-literal=CRONITOR_API_KEY=<api key>
```

Deploy using Helm 2 or Helm 3, as in the following example:

```
helm upgrade --install <release name> . --namespace <namespace> \
    --set credentials.secretName=cronitor-secret \
    --set credentials.secretKey=CRONITOR_API_KEY
```

## Values
_ To be written_