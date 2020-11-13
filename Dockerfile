FROM golang:1.15-alpine AS build

RUN apk add --update git
ENV GO111MODULE=on CGO_ENABLED=0
WORKDIR /code/

COPY go.mod go.sum /code/
RUN go mod download

COPY . /code/
RUN go build -o /bin/cronitor-k8s

FROM scratch
COPY --from=build /bin/cronitor-k8s /bin/cronitor-k8s
ENTRYPOINT ["/bin/cronitor-k8s"]
