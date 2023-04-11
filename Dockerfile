# syntax=docker/dockerfile:1

FROM golang:1.19.5

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /picord

# Run
CMD ["/picord"]