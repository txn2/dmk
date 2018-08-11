FROM golang:1.10.3 as builder

ARG VERSION

RUN mkdir -p /go/src/github.com/txn2/dmk
COPY . /go/src/github.com/txn2/dmk

WORKDIR /go/src/github.com/txn2/dmk
RUN go get ./...
RUN go build -a -installsuffix cg -o dmk \
    -ldflags "-w -s -X github.com/txn2/dmk/cli.Version=${VERSION}" \
    ./dmk.go


FROM alpine:3.7
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/github.com/txn2/dmk/dmk /dmk

RUN mkdir -p /projects
ENTRYPOINT ["/dmk"]