FROM golang:1.11.1 as builder

ARG VERSION

RUN mkdir -p /go/src/github.com/txn2/dmk
COPY . /go/src/github.com/txn2/dmk

WORKDIR /go/src/github.com/txn2/dmk

RUN GOOS=linux go build -a -installsuffix cg -o dmk \
    -ldflags "-w -s -X github.com/txn2/dmk/cli.Version=${VERSION}" \
    ./dmk.go

FROM ubuntu:18.04
WORKDIR /
COPY --from=builder /go/src/github.com/txn2/dmk/dmk /dmk

ENTRYPOINT ["/dmk"]