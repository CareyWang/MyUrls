FROM golang:1.13-alpine AS dependencies 
WORKDIR /app
RUN go env -w GO111MODULE="on" && go env -w GOPROXY="https://goproxy.cn,direct"

COPY go.sum go.mod ./
RUN go mod tidy 

FROM dependencies as build
WORKDIR /app
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myurls main.go 

FROM scratch
WORKDIR /app
COPY --from=build /app/myurls ./
EXPOSE 8002
ENTRYPOINT ["/app/myurls"]
