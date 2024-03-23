FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .

# RUN go env -w GOPROXY=https://mirrors.cloud.tencent.com/go/,direct
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o myurls

FROM scratch
WORKDIR /app
COPY --from=build /app/myurls ./
COPY public/* ./public/
EXPOSE 8080
ENTRYPOINT ["/app/myurls"]
