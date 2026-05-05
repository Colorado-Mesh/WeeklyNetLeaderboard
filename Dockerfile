FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /meshmonday ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=build /meshmonday /usr/local/bin/meshmonday
COPY web ./web
RUN mkdir -p /app/data
EXPOSE 8080
CMD ["meshmonday"]
