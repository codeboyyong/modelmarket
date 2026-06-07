FROM golang:1.25-alpine AS build

WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
RUN go build -o /bin/model-market-server ./cmd/server

FROM alpine:3.20

RUN adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=build /bin/model-market-server /bin/model-market-server
USER appuser

ENV HTTP_ADDR=:8080

EXPOSE 8080
CMD ["/bin/model-market-server"]
