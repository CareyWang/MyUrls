FROM ubuntu as download 
RUN apt-get update \
&& apt-get install -y git \
&& git clone https://github.com/CareyWang/MyUrls \
&& cp MyUrls/go.sum ./ \
&& cp MyUrls/go.mod ./ \
&& cp MyUrls/main.go ./ \
&& cp -r MyUrls/public ./ \
&& rm -rf MyUrls \
&& apt-get purge --auto-remove -y git 

FROM golang:1.15-alpine AS dependencies 
WORKDIR /app
RUN go env -w GO111MODULE="on" && go env -w GOPROXY="https://goproxy.cn,direct"

COPY --from=download go.sum go.mod ./
RUN go mod tidy 

FROM dependencies as build
WORKDIR /app
COPY --from=download main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myurls main.go 

FROM nginx
ADD init /init.sh
RUN chmod +x /init.sh
ADD nginx.conf /nginx.conf
COPY --from=build /app/myurls ./
COPY --from=download public/* ./public/
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
ENTRYPOINT ["./init.sh"]
