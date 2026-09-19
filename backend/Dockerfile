FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /pulsepoll-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /pulsepoll-api /pulsepoll-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/pulsepoll-api"]
