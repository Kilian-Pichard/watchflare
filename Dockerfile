# Stage 1: Build frontend
FROM node:24-alpine AS frontend-builder
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
RUN mkdir -p /app/data/pki

# Stage 3: Runtime
FROM dhi.io/debian-base:trixie
LABEL org.opencontainers.image.source="https://github.com/Kilian-Pichard/watchflare"
LABEL org.opencontainers.image.description="Watchflare Server Monitoring"
COPY --from=backend-builder --chown=65532:65532 /app/backend/watchflare-backend /usr/local/bin/watchflare-backend
COPY --from=backend-builder --chown=65532:65532 /app/data /var/lib/watchflare
USER 65532
VOLUME ["/var/lib/watchflare"]
EXPOSE 8080 50051
CMD ["watchflare-backend"]
