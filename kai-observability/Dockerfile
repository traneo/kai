FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY observability /app/
WORKDIR /app
EXPOSE 8082
ENTRYPOINT ["/app/observability"]
