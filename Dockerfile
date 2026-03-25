FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -tags "fts5" -ldflags="-X main.version=2.0.0" -o /lionclaw ./cmd/lionclaw

FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite-libs
COPY --from=builder /lionclaw /usr/local/bin/lionclaw
VOLUME /data
ENV LIONCLAW_DATA_DIR=/data
EXPOSE 18790
ENTRYPOINT ["lionclaw"]
CMD ["start"]
