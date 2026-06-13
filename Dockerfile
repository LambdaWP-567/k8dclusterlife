# Stage 1: Build React frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web/app
COPY web/app/package*.json ./
RUN npm ci
COPY web/app/ ./
RUN npm run build

FROM node:22-alpine AS testdash-builder
WORKDIR /app/web/testdash
COPY web/testdash/package*.json ./
RUN npm ci
COPY web/testdash/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# Stage 3: Final image
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/bin/server ./server
COPY --from=frontend-builder /app/web/app/dist ./web/app/dist
COPY --from=testdash-builder /app/web/testdash/dist ./web/testdash/dist
EXPOSE 8080
USER nobody
ENTRYPOINT ["./server"]
