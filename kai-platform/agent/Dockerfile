FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /agent ./cmd/agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git

# kai CLI must be installed in this image separately.
# For production, build the kai binary and copy it here:
# COPY --from=kai-build /app/publish/kai /usr/local/bin/kai

COPY --from=builder /agent /agent
EXPOSE 50052
ENTRYPOINT ["/agent"]
