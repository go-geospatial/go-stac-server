FROM golang:alpine AS builder
WORKDIR /go/src
COPY ./ .
RUN go build

FROM alpine
WORKDIR /usr/local

COPY --from=builder /go/src/go-stac-server /usr/local
ENTRYPOINT ["/usr/local/go-stac-server"]
CMD ["serve"]
