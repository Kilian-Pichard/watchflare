# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build backend (with embedded frontend)
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY shared/ ./shared/
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY backend/ ./backend/
COPY --from=frontend-builder /app/frontend/build ./backend/frontend/dist
RUN cd backend && CGO_ENABLED=0 go build -tags embed_frontend -o watchflare-backend

# Stage 3: Runtime
FROM alpine:3.21
LABEL org.opencontainers.image.source="https://github.com/Kilian-Pichard/watchflare"
LABEL org.opencontainers.image.description="Watchflare Server Monitoring"
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /app/backend/watchflare-backend /usr/local/bin/watchflare-backend
RUN mkdir -p /var/lib/watchflare/pki
EXPOSE 8080 50051
CMD ["watchflare-backend"]
