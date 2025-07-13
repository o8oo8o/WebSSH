# 指定创建的基础镜像
FROM alpine:3.21.0
  
# 替换阿里云的并更新源、安装openssh 并修改配置文件和生成key 并且同步时间
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \    
	&& apk update \    
	&& apk add --no-cache openssh openssh-server tzdata \
	&& cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
	&& sed -i "s/#PermitRootLogin.*/PermitRootLogin yes/g" /etc/ssh/sshd_config \
	&& sed -i "s/#PasswordAuthentication.*/PasswordAuthentication yes/g" /etc/ssh/sshd_config \
	&& ssh-keygen -t dsa -P "" -f /etc/ssh/ssh_host_dsa_key \
	&& ssh-keygen -t rsa -P "" -f /etc/ssh/ssh_host_rsa_key \ 
	&& ssh-keygen -t ecdsa -P "" -f /etc/ssh/ssh_host_ecdsa_key \
	&& ssh-keygen -t ed25519 -P "" -f /etc/ssh/ssh_host_ed25519_key \
	&& echo "root:admin" | chpasswd

# 开放22端口
EXPOSE 22
 
# 容器启动时执行ssh启动命令
CMD ["/usr/sbin/sshd", "-D"]
