
.PHONY build:
	DOCKER_BUILDKIT=1 docker build -t cronitor-k8s:latest .

.PHONY push:
	#docker tag cronitor-k8s:latest 123849920427.dkr.ecr.us-east-1.amazonaws.com/cronitor-k8s:latest
	docker tag cronitor-k8s:latest jdotjdot/cronitor-k8s:latest
	docker push jdotjdot/cronitor-k8s:latest

.PHONY docker:
	make build
	make push

.PHONY helm:
	helm3 upgrade --install -n cronitor cronitor-k8s ./charts/cronitor-k8s-agent/