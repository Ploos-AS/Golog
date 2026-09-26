FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/golog ./cmd/golog

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S golog \
    && adduser -S -G golog golog \
    && mkdir -p /app/rules /app/data \
    && chown -R golog:golog /app
WORKDIR /app
COPY --from=build /out/golog /usr/local/bin/golog
COPY --chown=golog:golog rules ./rules
USER golog
ENV GOLOG_RULES=/app/rules/hello.pl \
    GOLOG_STATE=/app/data/state.json
ENTRYPOINT ["/usr/local/bin/golog"]
