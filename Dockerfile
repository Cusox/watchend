FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /watchend ./cmd/watchend

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && addgroup -S watchend && adduser -S -G watchend watchend && mkdir /data && chown watchend:watchend /data
COPY --from=build /watchend /usr/local/bin/watchend
USER watchend
VOLUME ["/data"]
EXPOSE 3000
ENTRYPOINT ["watchend"]
