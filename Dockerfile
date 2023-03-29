FROM golang:1.20-alpine AS build
WORKDIR /app
COPY main.go go.mod go.sum .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o myurls main.go
RUN apk update && apk add upx
RUN upx myurls

FROM scratch
WORKDIR /app
COPY --from=build /app/myurls ./
COPY public/* ./public/
EXPOSE 8002
ENTRYPOINT ["/app/myurls"]
