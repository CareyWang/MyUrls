FROM golang:1.20-alpine AS build
RUN apk update && apk add upx
WORKDIR /app
COPY main.go go.mod go.sum .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o myurls main.go \
    && upx myurls

FROM scratch
WORKDIR /app
COPY --from=build /app/myurls ./
COPY public/* ./public/
EXPOSE 8002
ENTRYPOINT ["/app/myurls"]
