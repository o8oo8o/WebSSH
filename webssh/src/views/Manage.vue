<template>
  <div v-if="is_login">
    <el-tabs v-model="active_name" type="card">
      <el-tab-pane label="连接状态" name="connectInfo">
        <el-table :data="host_list" style="width: 100%">
          <el-table-column fixed prop="ip" label="地址" width="250">
          </el-table-column>
          <el-table-column prop="username" label="用户名" width="180">
          </el-table-column>
          <el-table-column prop="port" label="端口"> </el-table-column>
          <el-table-column prop="shell" label="shell"> </el-table-column>

          <el-table-column prop="session_id" label="会话ID" width="200">
          </el-table-column>

          <el-table-column prop="start_time" label="连接创建时间" width="150">
          </el-table-column>

          <el-table-column fixed="right" label="操作">
            <template #default="scope">
              <el-button
                size="mini"
                type="danger"
                @click="disconnect(scope.$index, scope.row)"
                >断开</el-button
              >
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="管理密码" name="second">
        <el-container>
          <el-form label-width="100px">
            <el-form-item label="旧密码">
              <el-input
                type="password"
                minlength="1"
                maxlength="64"
                v-model.trim="old_password"
              ></el-input>
            </el-form-item>
            <el-form-item label="新密码">
              <el-input
                type="password"
                minlength="1"
                maxlength="64"
                v-model.trim="new_password_a"
              ></el-input>
            </el-form-item>
            <el-form-item label="新密码">
              <el-input
                type="password"
                minlength="1"
                maxlength="64"
                v-model.trim="new_password_b"
              ></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="changePassword">提交</el-button>
            </el-form-item>
          </el-form>
        </el-container>
      </el-tab-pane>
    </el-tabs>
  </div>
  <div v-else>
    <el-container>
      <el-form label-width="80px" class="login-box">
        <h3 class="login-title">请登录</h3>
        <el-form-item label="密码">
          <el-input
            type="password"
            placeholder="请输入密码"
            v-model="password"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="login">登录</el-button>
        </el-form-item>
      </el-form>
    </el-container>
  </div>
</template>


<script lang="ts">
import "xterm/css/xterm.css";

import {
  computed,
  defineComponent,
  nextTick,
  onMounted,
  reactive,
  ref,
  toRefs,
  watch,
  watchEffect,
} from "vue";
import xhttp, { Xhttp } from "../xhttp";
import { Terminal } from "xterm";
import { AttachAddon } from "xterm-addon-attach";
import { FitAddon } from "xterm-addon-fit";
import { ElMessage, ElMessageBox, ElPopover } from "element-plus";
import { Router, useRoute, useRouter } from "vue-router";

let route: Router;

enum Mode {
  "create" = 0,
  "update" = 1,
}

interface HostInfo {
  ip: string;
  username: string;
  port: number;
  session_id: string;
  shell: string;
  start_time: string;
}

let data = reactive({
  active_name: "connectInfo",
  host_list: [] as Array<HostInfo>,
  is_login: false,
  password: "",

  old_password: "",
  new_password_a: "",
  new_password_b: "",

});

/**
 * 断开链接
 */
function disconnect(index: number, row: HostInfo) {
  let xhttp = new Xhttp();
  xhttp
    .delete(`/api/status?session_id=${row.session_id}`)
    .then((response) => response.json())
    .then((json) => {
      if (json.code === 0) {
        getStatus();
      }
    });
}

function getStatus() {
  let xhttp = new Xhttp();
  xhttp
    .get("/api/status")
    .then((response) => {
      if (response.status === 401) {
        // alert("没有登陆");
        data.is_login = false;
      }
      return response.json();
    })
    .then((json) => {
      if (json.code === 0) {
        data.is_login = true;
        data.host_list = json.data;
      }
    });
}

function login() {
  let fm = new FormData();
  fm.append("pwd", data.password);
  let xhttp = new Xhttp();
  xhttp
    .post("/api/login", { body: fm })
    .then((response) => {
      if (response.status === 401) {
        ElMessage.error("密码错误");
        return;
      }
      return response.json();
    })
    .then((json) => {
      if (json.code === 0) {
        data.is_login = true;
        getStatus();
      }
    });
}

function changePassword() {
  if (data.old_password.length === 0) {
    ElMessage.error("旧密码不能为空");
    return;
  }

  if (data.new_password_a.length === 0) {
    ElMessage.error("新密码不能为空");
    return;
  }

  if (data.new_password_b.length === 0) {
    ElMessage.error("确认新密码不能为空");
    return;
  }

  if (data.new_password_a !== data.new_password_b) {
    ElMessage.error("两次密码输入不一致");
    return;
  }

  let fm = new FormData();
  fm.append("old_pwd", data.old_password);
  fm.append("new_pwd", data.new_password_a);
  let xhttp = new Xhttp();
  xhttp
    .patch("/api/login", { body: fm })
    .then((response) => response.json())
    .then((json) => {
      if (json.code === 0) {
        ElMessage.success("密码修改成功");
        data.is_login = false;
      }
    });
}

// 报告连接状态
function refreshConnectInfo() {
  if (data.is_login) {
    getStatus();
    setInterval(getStatus, 10000);
  }
}

export default defineComponent({
  name: "manage",

  setup(props: any, context) {
    return {
      ...toRefs(data),
      login,
      disconnect,
      changePassword,
    };
  },
});
</script>


<style scoped>
.login-box {
  border: 1px solid #dcdfe6;
  width: 350px;
  margin: 180px auto;
  padding: 35px 35px 15px 35px;
  border-radius: 5px;
  -webkit-border-radius: 5px;
  -moz-border-radius: 5px;
  box-shadow: 0 0 25px #909399;
}
.login-title {
  text-align: center;
  margin: 0 auto 40px auto;
  color: #303133;
}
</style>