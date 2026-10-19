FROM docker.io/library/node:22-alpine AS web-builder
WORKDIR /web
COPY web/package*.json ./
RUN sed -i '/"resolved":/d' package-lock.json 2>/dev/null || true
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM docker.io/library/golang:1.24-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/go-ls-server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/go-ls .

FROM docker.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S gols && adduser -S -G gols gols

COPY --from=go-builder /out/go-ls-server /usr/local/bin/go-ls-server
COPY --from=go-builder /out/go-ls /usr/local/bin/go-ls
COPY --from=web-builder /web/out /usr/local/share/go-ls/web

USER gols
EXPOSE 8080
ENV GOLS_LISTEN_ADDR=":8080"
ENV GOLS_UI_DIR="/usr/local/share/go-ls/web"
ENTRYPOINT ["/usr/local/bin/go-ls-server"]
