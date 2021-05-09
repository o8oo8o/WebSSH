# GoWebSSH
<br/>

### 介绍：
* Golang + Vue 实现一个Web版单文件的SSH管理工具
* 借助于Golang embed,打包以后只有一个文件,简单高效
<br/>
<br/>

### 在线Demo：
* [点我](https://www.huangrui.vip:12345)
<br/>
<br/>


### 目标：&nbsp;&nbsp;取代Xshell
* 目前虽然只实现xshell部分功能,未来计划逐步更新
<br/>
<br/>

### 主要功能：
* 支持同时连接多个主机
* 可以保存主机连接信息
* 终端窗口大小根据浏览器窗口自适应
* 支持直接通过Web上传下载文件
* 支持自定义终端字体大小、字体颜色、字体样式
* 支持自定义背景、光标颜色及光标样式
* 支持后台管理,强制断开连接
* 已保存的主机信息可直接编辑并连接

<br/>
<br/>

### 后端介绍：
* 使用最新Golang 1.16版本实现后端功能
* 实现配置文件读取功能
* 基于内存的session功能
* 借助于sqlite可把主机信息持久化
<br/>
<br/>


### 前端介绍：
* 使用最新版Vue3 + TypeScript实现前端逻辑
* 前端UI使用最近element-plus(目前还没有稳定版)
* 基于最新版xterm.js + Websocket 实现终端
<br/>
<br/>

---
### 使用说明：
* git clone https://github.com/o8oo8o/GoWebSSH.git


---
### 演示截图
![avatar](./img/a.png)
![avatar](./img/b.png)
![avatar](./img/c.png)
![avatar](./img/d.png)
![avatar](./img/e.png)








