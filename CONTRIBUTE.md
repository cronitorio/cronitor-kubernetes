Contributing
======

To set up a development environment:
1. Ensure Go is installed, maybe update to the latest
2. Install [Kind](https://kind.sigs.k8s.io/), [Skaffold](https://skaffold.dev/) and [Helm](https://helm.sh/docs/intro/install)
3. Run `kind create cluster` to create a new Kubernetes cluster locally
4. Add a `Secret` object to the cluster containing your Cronitor API key according to the [instructions](./README.md#instructions)
5. In the main directory of this repository, run `skaffold dev`. Skaffold will build the Docker container and agent and push it to your local Kubernetes cluster. Anytime you make code changes it will rebuild and re-push.
6. If you'd like to add some sample `CronJob`s easily, you can run `kubectl apply -f e2e-test/resources/main/`, which will add a bunch of pre-written test resources.
7. When you're done, stop skaffold and clean up with `kind delete cluster`!


To make a new release:
1. Update the `version` number in the chart's `Chart.yaml`
2. Update the `appVersion` number if necessary -- if any changes were made to the application itself.
3. Push to the `main` branch
4. Profit
