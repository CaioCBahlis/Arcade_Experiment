# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tetris .

FROM alpine:3.20
RUN adduser -D -u 10001 tetris
USER tetris
# Bubble Tea picks colors from TERM; fall back to something widely supported.
ENV TERM=xterm-256color
COPY --from=build /out/tetris /usr/local/bin/tetris
ENTRYPOINT ["tetris"]
