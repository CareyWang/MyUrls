# MyUrls

基于 golang1.15 与 Redis 实现的本地短链接服务，用于缩短请求链接与短链接还原。

## Table of Contents

- [Update](#update)
- [Dependencies](#dependencies)
- [Docker](#Docker)
- [Install](#install)
- [Usage](#usage)
- [API](#api)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [License](#license)

# Update

- 20200330
  集成前端至根路径，如: <http://127.0.0.1:8002/>。

  > 注：如需使用集成的前端，项目部署请 clone 仓库后自行编译，并在代码根目录启动服务。或者可 nginx 单独配置 root 至 public 目录的 index.html。


#1 安装
1.1 安装 MyURLs 后端服务
MyURLs – Github 作者主页 ，上面有docker 和 docker-compose 的安装代码，也可以前往 Release 下载对应平台可执行文件。

可以使用宝塔文件管理的远程下载，或者下载到本地，再传到服务器。

这里服务器目录以/home/为例，把下载的压缩包传到/home/目录进行解压，接着启动终端SSH登录，执行以下两行命令

（example.com 即返回的短链域名，不需要”http(s)://”）

cd /home/myurls
./linux-amd64-myurls.service -domain example.com
执行完，什么都没有返回，处于一个监视状态，此时应该是临时启动服务了。

linux-amd64-myurls.service 必须带 -domain 参数启动，其它参数可以不带。不带的即是默认的，具体参数如下：

./build/linux-amd64-myurls.service -h

Usage of ./build/linux-amd64-myurls.service:
-conn string
      Redis连接，格式: host:port (default "127.0.0.1:6379")

-domain string
      短链接域名，必填项

-passwd string
      Redis连接密码

-port int
      服务端口 (default 8002)

-ttl int
      短链接有效期，单位(天)，默认90天。 (default 90)
1.2 安装 MyURLs 前端WEB
登录宝塔的文件管理，编辑 /home/myurls/public/index.html 文件，将 const backend 这里改成自己的域名，然后保存。

const backend = 'https://example.com
如果你前端web需要开启SSL那么这里也要写成 https，强烈建议开启SSL。

然后宝塔面板 “网站”-“添加站点”，使用上面的域名建一个站，站点根目录设置为 /home/myurls/public，并开启SSL。接着点下面的“反向代理”，填入后端服务地址和端口，提交即可。

6cf6dfcf99acd8e
此时使用浏览打开你的域名就可以看到前端界面了。

1.3 添加 MyURLs 自启动
新建文件 /etc/systemd/system/myurls.service，写入内容然后保存：（ExecStart 、WorkingDirectory两个参数后面的路径改成自己的网站根目录，以及将 example.com 替换为自己的网址）

[Unit]
Description=A API For Short URL Convert
After=network.target

[Service]
Type=simple
ExecStart=/home/myurls/linux-amd64-myurls.service -domain example.com
WorkingDirectory=/home/myurls
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
接着终端 Ctrl+C 退出 linux-amd64-myurls.service。

1.4 完成配置及开机启动
更新配置
systemctl daemon-reload

启动服务
systemctl start myurls

设置开机启动
systemctl enable myurls
以下为日常管理命令：

启动服务
systemctl start myurls

停止服务
systemctl stop myurls

重启服务
systemctl restart myurls

查看状态
systemctl status myurls
如果不需要从其他网域调用的话，这就完成了，如果需要从其他域使用，请继续。

2 其他网域调用
如果不加其他配置直接从其他域调用MyURLs会出现不能返回结果的错误，但是MyURLs后端实际是收到请求并且生成了短链代码，但就是不能返回到其他域。比如MyURLs作者的 Sub-web项目。这里我们需要网站设置的 Nginx反向代理配置文件 第23行后面添加 add_header Access-Control-Allow-Origin *;

    set $static_fileXeorKt9o 0;
    if ( $uri ~* "\.(gif|png|jpg|css|js|woff|woff2)$" )
    {
    	set $static_fileXeorKt9o 1;
    	expires 12h;
        }
    if ( $static_fileXeorKt9o = 0 )
    {
    add_header Cache-Control no-cache;
    add_header Access-Control-Allow-Origin *;
    }
}
保存即可，这样就可以跨域使用了。


## Maintainers

[@CareyWang](https://github.com/CareyWang)

## Contributing

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © 2020 CareyWang
