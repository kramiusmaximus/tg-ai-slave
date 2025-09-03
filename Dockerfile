# Builder stage for both development and production
FROM golang:1.25 AS builder
WORKDIR /openrouter-bot
RUN go install github.com/air-verse/air@latest
COPY . .
RUN go mod download
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /openrouter-bot/openrouter-bot

# Final production image
FROM alpine:3.22 AS production
WORKDIR /openrouter-bot
COPY --from=builder /openrouter-bot/config.yaml ./
COPY --from=builder /openrouter-bot/lang/ ./lang/
COPY --from=builder /openrouter-bot/openrouter-bot ./
RUN mkdir logs