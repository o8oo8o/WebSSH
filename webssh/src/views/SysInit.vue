<template>

  <div>
    <el-container>
      <el-header>
        <h1 style="text-align: center;">系统初始化</h1>
      </el-header>
      <el-main>
        <el-row>
          <el-col :span="4"></el-col>
          <el-col :span="16">
            <div>
              <el-steps :active="active" finish-status="success">
                <el-step title="数据库选择" />
                <el-step title="Web账号配置" />
                <el-step title="SSH服务器配置" />
                <el-step title="完成" />
              </el-steps>

              <el-divider />
              <el-form :model="form">

                <div v-if="active === 0">
                  <el-form-item label="数据库">
                    <el-radio-group v-model="form.db_type">
                      <el-radio value="sqlite">SQLite</el-radio>
                      <el-radio value="mysql">MySQL</el-radio>
                      <el-radio value="pgsql">PostgresSQL</el-radio>
                    </el-radio-group>
                  </el-form-item>

                  <el-form-item v-if="form.db_type === 'sqlite'">
                    <el-card style="width: 100%;">
                      <template #header>
                        <div class="card-header">
                          <span>样例:&nbsp;&nbsp;gowebssh.db</span>
                        </div>
                      </template>

                      <el-input v-model="form.sqlite_dbdsn" minlength="10" maxlength="65535" show-word-limit clearable
                        placeholder="请输入Sqlite数据库连接配置,参考上面的样例">
                      </el-input>
                    </el-card>
                  </el-form-item>

                  <el-form-item v-if="form.db_type === 'mysql'">
                    <el-card style="width: 100%;">
                      <template #header>
                        <div class="card-header">
                          <span>样例:&nbsp;&nbsp;root:mypwd@tcp(192.168.150.200:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local</span>
                        </div>
                      </template>

                      <el-input v-model="form.mysql_dbdsn" minlength="10" maxlength="65535" show-word-limit clearable
                        placeholder="请输入MySQL数据库连接配置,参考上面的样例">
                      </el-input>
                    </el-card>
                  </el-form-item>

                  <el-form-item v-if="form.db_type === 'pgsql'">
                    <el-card style="width: 100%;">
                      <template #header>
                        <div class="card-header">
                          <span>样例:&nbsp;&nbsp;host=192.168.150.200 port=5432 user=postgres dbname=mydb sslmode=disable
                            password=mypwd</span>
                        </div>
                      </template>

                      <el-input v-model="form.pgsql_dbdsn" minlength="10" maxlength="65535" show-word-limit clearable
                        placeholder="请输入PostgresSQL数据库连接配置,参考上面的样例">
                      </el-input>
                    </el-card>
                  </el-form-item>
                </div>

                <div v-if="active === 1">
                  <div>
                    <el-form-item label="Web登录账号:">
                      <el-input v-model="form.name" trim minlength="1" maxlength="64" show-word-limit clearable
                        placeholder="请输入用户名">
                      </el-input>
                    </el-form-item>
                    <el-form-item label="Web登录密码:">
                      <el-input v-model="form.pwd" trim type="password" minlength="3" maxlength="64" show-word-limit
                        show-password clearable placeholder="请输入密码">
                      </el-input>
                    </el-form-item>
                  </div>
                </div>

                <div v-if="active === 2">
                  <div>
                    <el-form-item label="SSH服务器登录账号:">
                      <el-input v-model="form.sshd_user" trim minlength="1" maxlength="64" show-word-limit clearable
                        placeholder="请输入账号">
                      </el-input>
                    </el-form-item>
                    <el-form-item label="SSH服务器登录密码:">
                      <el-input v-model="form.sshd_pwd" trim type="password" minlength="3" maxlength="64"
                        show-word-limit show-password clearable placeholder="请输入密码">
                      </el-input>
                    </el-form-item>
                    <el-form-item label="SSH服务器监听端口:">
                      <el-input-number v-model="form.sshd_port" :min="1" :max="65535" />
                    </el-form-item>
                    <el-form-item label="SSH服务器监听地址:">
                      <el-select v-model="form.sshd_host" placeholder="选择监听地址">
                        <el-option label="0.0.0.0" value="0.0.0.0" />
                        <el-option label="127.0.0.1" value="127.0.0.1" />
                      </el-select>
                    </el-form-item>
                  </div>
                </div>

                <div v-if="active === 3">
                  <el-form-item>
                    <el-card style="width: 100%;">
                      <template #header>
                        <div class="card-header">
                          <span style="color: red;">配置确认,请妥善保存密码</span>
                        </div>
                      </template>
                      <p>数据库类型:&nbsp;&nbsp;{{ db_kind }}</p>
                      <p>数据库DSN:&nbsp;&nbsp;{{ db_dsn }}</p>
                      <p>Web登录账号:&nbsp;&nbsp;{{ form.name }}</p>
                      <p>Web登录密码:&nbsp;&nbsp;{{ form.pwd }}</p>
                      </br>
                      <p>SSH服务器登录账号:&nbsp;&nbsp;{{ form.sshd_user }}</p>
                      <p>SSH服务器登录密码:&nbsp;&nbsp;{{ form.sshd_pwd }}</p>
                      <p>SSH服务器监听端口:&nbsp;&nbsp;{{ form.sshd_port }}</p>
                      <p>SSH服务器监听地址:&nbsp;&nbsp;{{ form.sshd_host }}</p>
                    </el-card>
                  </el-form-item>
                </div>

                <div>
                  <el-form-item>
                    <el-button type="success" v-if="active === 0" @click="dbConnCheck">测试连接</el-button>
                    <el-button v-if="active > 0" type="primary" @click="goPrev">上一步</el-button>
                    <el-button v-if="active < 3" type="primary" @click="goNext">下一步</el-button>
                    <el-button v-if="active === 3" type="success" @click="sysInit">完成</el-button>

                    &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
                    <div v-if="active === 3">
                      <el-popconfirm confirmButtonText="确定" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                        title="确定初始化" @confirm="sysInit">
                        <template #reference>
                          &nbsp;&nbsp;<el-button type="success">完成</el-button>
                        </template>
                      </el-popconfirm>
                    </div>

                  </el-form-item>
                </div>
              </el-form>
            </div>
          </el-col>
        </el-row>
      </el-main>
    </el-container>
  </div>

</template>

<script lang="ts" setup>
import {
  computed,
  onBeforeMount,
  reactive,
  ref,
} from "vue";

import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import axios from "axios";
import { useGlobalStore } from "@/stores/store";


let globalStore = useGlobalStore();
let router = useRouter();


const form = reactive({
  sqlite_dbdsn: "gowebssh.db",
  mysql_dbdsn: "",
  pgsql_dbdsn: "",
  db_type: "sqlite",
  name: "",
  pwd: "",
  sshd_host: "0.0.0.0",
  sshd_port: 22622,
  sshd_user: "",
  sshd_pwd: ""
})

let db_dsn = computed<string>(() => {
  if (form.db_type == "mysql") {
    return form.mysql_dbdsn
  }
  if (form.db_type == "pgsql") {
    return form.pgsql_dbdsn
  }
  if (form.db_type == "sqlite") {
    return form.sqlite_dbdsn
  }
  return "";
})

let db_kind = computed<string>(() => {
  if (form.db_type == "mysql") {
    return "MySQL"
  }
  if (form.db_type == "pgsql") {
    return "PostgreSQL"
  }
  if (form.db_type == "sqlite") {
    return "SQLite"
  }
  return "";
})


const active = ref<number>(0)

const goPrev = () => {
  if (active.value-- < 0) {
    active.value = 0

  }
}
const goNext = () => {
  if (active.value++ > 2) {
    active.value = 0
  }
}

interface ResponseData {
  code: number;
  msg: string;
  data?: any
}

function dbConnCheck() {
  let str = db_dsn.value.trim();
  if (str.length < 2) {
    ElMessage.error("请输入正确配置")
    return;
  }
  axios.post<ResponseData>(`/api/sys/db_conn_check`,
    { "db_type": form.db_type, "db_dsn": str }
  ).then((ret) => {
    if (ret.data.code === 0) {
      ElMessage.success("连接数据库成功");
    } else {
      ElMessage.error("连接数据库错误:" + ret.data.msg)
    }
  }).catch((ret) => {
    ElMessage.error("连接数据库错误异常");
  })
}

function randomString(length: number) {
  let result = "";
  const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const charactersLength = characters.length;

  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }

  return result;
}

function sysInit() {
  let initData = {
    "db_type": form.db_type,
    "db_dsn": db_dsn.value.trim(),
    "jwt_secret": randomString(64),
    "session_secret": randomString(64),
    "username": form.name.trim(),
    "password": form.pwd.trim(),
    "sshd_host": form.sshd_host.trim(),
    "sshd_port": form.sshd_port,
    "sshd_user": form.sshd_user.trim(),
    "sshd_pwd": form.sshd_pwd.trim()
  }
  if (initData.db_dsn.length < 5) {
    ElMessage.error("系统初始化错误:请输入正确的数据库连接")
    return
  }

  if (initData.username.length < 2) {
    ElMessage.error("系统初始化错误:Web登录账号至少两个字符")
    return
  }

  if (initData.password.length < 2) {
    ElMessage.error("系统初始化错误:Web登录密码至少两个字符")
    return
  }

  if (initData.sshd_user.length < 2) {
    ElMessage.error("系统初始化错误:SSH服务器登录账号至少两个字符")
    return
  }

  if (initData.sshd_pwd.length < 2) {
    ElMessage.error("系统初始化错误:SSH服务器登录密码至少两个字符")
    return
  }

  axios.post<ResponseData>(`/api/sys/init`, initData)
    .then((ret) => {
      if (ret.data.code === 0) {
        ElMessage.success("系统初始化成功");
        globalStore.isInit = true;
        router.push({ "name": "Login" });
      } else {
        ElMessage.error("系统初始化错误:" + ret.data.msg)
      }
    }).catch((ret) => {
      console.log(ret)
      ElMessage.error("系统初始化错误异常");
    })
}

onBeforeMount(() => {
  if (globalStore.isInit) {
    router.push({ "name": "Login" })
  }
})

</script>

<style scoped></style>
