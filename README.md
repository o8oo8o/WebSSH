## GoWebSSH
<br/>

### 项目介绍：
* **Web版ssh客户端 + (sshd,sftp)服务端实现**
<br/>

### 概要：
* Golang 1.23 + (Vue3.5 + Vite6)  实现一个Web版单文件的(SSH+SSHD)
* 借助于Golang embed,打包以后只有一个文件,简单高效
* 使用及编译过程,超级简单,绝对保姆级
* 上一版主要本地运行,但是通过部分用户反馈,此项目定位改为服务器运行,所以此版本加入了很多企业场景中的功能
<br/>

### 联系我：
* **QQ:774309635**
<br/>

---
### Quick start(大象装进冰箱只需3步)：
>  必须使用golang 1.21以上版本
* git clone https://github.com/o8oo8o/WebSSH.git
* cd WebSSH/gossh
* go build
* ./gossh
>  打开链接 http://127.0.0.1:8899/ 开始享用吧,第一次需要初始化
<br/>

### Docker 方式：
* git clone https://github.com/o8oo8o/WebSSH.git
* cd WebSSH
* docker build -f Dockerfile -t gowebssh:v2 .
* docker run -d --name webssh -p 8899:8899 -v gowebssh:/var/lib/webssh gowebssh:v2
<br/>

### 打赏我：
* **每一个开源项目的背后，都有一群默默付出、充满激情的开发者。他们用自己的业余时间，不断地优化代码、修复bug、撰写文档，只为让项目变得更好。如果您觉得我的项目对您有所帮助，如果您认可我的努力和付出，那么请考虑给予我一点小小的打赏，够买一瓶啤酒就行🍺，如果能同时打赏啤酒花生那更好🍺🥜，因为所有的代码都是喝完酒撸的。放上收款码的时候我是羞愧的，一个中年男人的最后的尊严和节操竟然没了😂，友情提示:打赏不退，怕被媳妇查到大额支出🥸，如果需要技术支持，需要收费哦**
<br/>
<br/>

![打赏二维码](https://gitee.com/o8oo8o/WebSSH/raw/main/img/pay.png)

<br/>

### 运行环境依赖：
* 需要MySQL8+及PostgreSQL12.2+或者直接使用内置SQLite数据库

### SSHD服务器功能：
* 可以配置只监听本地端口
* 支持Web配置sshd服务器账号密码及公钥
* 支持密码认证,公钥认证,增强登录过程的安全性
* 通过SFTP或SCP用户可以安全地在本地和远程服务器之间传输文件
<br/>

### Web客户端主要功能：
* 支持同时连接多个主机,支持重连、清屏功能
* 支持IPv4、IPv6
* 支持SSH证书登陆及证书密码
* 支持批量支持命令,当前终端及所有终端
* 支持命令收藏,方便重复执行命令,批量发送命令到所有会话
* 可以保存主机连接信息
* 支持直接通过Web上传下载文件
* 支持直接通过Web创建目录,删除文件及目录功能
* 支持手动输入路径
* 支持自定义终端字体大小、字体颜色、字体样式
* 支持自定义背景、光标颜色及光标样式
* 已保存的主机信息可直接编辑并连接
* 支持后台管理,强制断开连接
* 支持登陆日志审计,方便监控违规操作
* 支持访问控制,在公网场景中有效拦截非法访问
<br/>

### 为什么这么简单:
* 为了方便您使用,把golang编译的依赖已经整理好了,clone就一起下载了
* 前端已经编译完成,并把编译完成的静态资源拷贝到gossh/webroot目录中
* 可执行文件内嵌静态资源,方便你随性所欲的移动可执行文件
* 因内置sshd服务器,在受限的网络环境依然能通过web访问
<br/>

### 配置文件：
* 第一次运行会在用户home目录创建一个 .GoWebSSH 目录
* GoWebSSH.toml 可以配置server端口等信息
* cert.pem HTTPS服务器证书文件
* key.key  HTTPS服务器私钥文件
<br/>

### 注意: 
* 当程序检测到cert.pem 和 key.key 文件,会使用https协议,否则使用http协议
* 用户只需把证书文件和私钥文件放到 .GoWebSSH 目录就可以了
```shell
openssl genpkey -algorithm RSA -out key.key -pkeyopt rsa_keygen_bits:2048

openssl req -new -x509 -key key.key -out cert.pem -days 365 -subj "/C=CN/ST=bj/L=bj/O=gowebssh/OU=gowebssh/CN=gowebssh.com"
```
<br/>

### Systemd 方式启动: 
```shell
cat > /etc/systemd/system/gowebssh.service << "END"
##################################
[Unit]
Description=GoWebSSH Daemon
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
Environment=TERM=xterm
Environment=XDG_SESSION_TYPE=tty
Environment=HOME=/root
PrivateTmp=true
LimitNOFILE=65535

# 执行程序路径
ExecStart=/usr/local/gossh

# auto restart
StartLimitIntervalSec=0
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
##################################
END

systemctl daemon-reload

systemctl start gowebssh.service

systemctl enable gowebssh.service

```
<br/>

---
### 演示截图：
![a](https://gitee.com/o8oo8o/WebSSH/raw/main/img/a.jpg)
![b](https://gitee.com/o8oo8o/WebSSH/raw/main/img/b.jpg)
![c](https://gitee.com/o8oo8o/WebSSH/raw/main/img/c.jpg)
![d](https://gitee.com/o8oo8o/WebSSH/raw/main/img/d.jpg)
![e](https://gitee.com/o8oo8o/WebSSH/raw/main/img/e.jpg)
![f](https://gitee.com/o8oo8o/WebSSH/raw/main/img/f.jpg)


