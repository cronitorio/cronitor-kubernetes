FROM golang:1.15-alpine AS build

RUN apk add --update git
ENV GO111MODULE=on CGO_ENABLED=0
WORKDIR /code/

# Precache gomod dependencies, so we don't need to redownload on every build
COPY go.mod go.sum /code/
RUN go mod download

COPY . /code/
RUN go build -o /bin/cronitor-kubernetes

FROM alpine:latest
COPY --from=build /bin/cronitor-kubernetes /bin/cronitor-kubernetes
ENTRYPOINT ["/bin/cronitor-kubernetes"]
