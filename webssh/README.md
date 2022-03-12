### 前端完整构建流程：
```
git clone https://github.com/o8oo8o/GoWebSSH.git

cd GoWebSSH/webssh/

npm install

npm run build

cd dist

cp -af * ../../gossh/webroot

cd ../../gossh  

go build

./gossh #启动

打开链接 http://127.0.0.1:8899/ 开始享用吧
```

