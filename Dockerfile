FROM node:20-alpine AS frontend
WORKDIR /build
COPY package.json tailwind.config.js ./
COPY web/static/css/input.css ./web/static/css/input.css
COPY web/templates ./web/templates
COPY web/static/js ./web/static/js
RUN npm install && \
    npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

FROM golang:1.22-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /build/web ./web
COPY --from=builder /build/internal/ai/prompts ./internal/ai/prompts
COPY --from=builder /build/migrations ./migrations
COPY --from=frontend /build/web/static/css/app.css ./web/static/css/app.css
COPY --from=builder /build/seed ./seed
RUN mkdir -p /app/data/uploads && chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080
CMD ["./server"]
