FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache make git

COPY go.mod go.sum Makefile ./
RUN make install

COPY . .

RUN make build

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin/valisgo /app/valisgo

EXPOSE 8080

CMD ["/app/valisgo"]
