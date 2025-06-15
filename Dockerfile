FROM golang:1.24-alpine AS build
WORKDIR /app
COPY . .

# RUN go env -w GOPROXY=https://mirrors.cloud.tencent.com/go/,direct
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o myurls ./cmd/myurls

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/myurls ./
COPY web/* ./web/

ENV MYURLS_STORAGE_TYPE=redis

EXPOSE 8080
ENTRYPOINT ["/app/myurls"]
