<template>
  <el-tab-pane label="连接状态" name="connectInfo">
    <el-card>
      <el-row>
        <el-table :data="data.host_list" style="width: 100%" :show-overflow-tooltip="true">
          <el-table-column fixed prop="name" label="名称"></el-table-column>
          <el-table-column fixed prop="address" label="服务器地址" width="150"></el-table-column>
          <el-table-column prop="port" label="端口" width="60"></el-table-column>
          <el-table-column prop="user" label="用户名"></el-table-column>
          <el-table-column prop="client_ip" label="客户端IP"></el-table-column>
          <el-table-column prop="session_id" label="会话ID" width="180"></el-table-column>
          <el-table-column prop="last_active_time" label="最后活跃时间" width="165"></el-table-column>
          <el-table-column prop="start_time" label="连接创建时间" width="165"></el-table-column>
          <el-table-column fixed="right" label="操作">
            <template #default="scope">
              <el-button type="danger" @click="disconnect(scope.$index, scope.row)">断开</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-row>
    </el-card>
  </el-tab-pane>
</template>

<script setup lang="ts">
import { onMounted, reactive } from "vue";
import axios from "axios";

/**
 * 主机信息
 */
interface HostInfo {
  id: number;
  name: string;
  address: string;
  user: string;
  auth_type: "pwd" | "cert";
  cert_data: string,
  cert_pwd: string
  pwd: string;
  port: number;
  font_size: number;
  background: string;
  foreground: string;
  cursor_color: string;
  font_family: string;
  cursor_style: "block" | "underline" | "bar";
  shell: string;
  session_id: string;
  start_time: string;
  last_active_time: string;
  client_ip: string;
}

interface ResponseData {
  code: number;
  msg: string;
  data?: any
}

let data = reactive({
  host_list: Array<HostInfo>(),
  is_admin: true,
});

interface OnlineInfo {
  code: number,
  msg: string,
  data: Array<HostInfo>
}

/**
 * 断开链接
 */
function disconnect(index: number, row: HostInfo) {
  try {
    axios.post(`/api/ssh/disconnect?session_id=${row.session_id}`);
  } catch (error) {
    console.log(error)
  }
}

/**
 * 获取在线客户端连接
 */
function getOnlineClient() {
  if (data.is_admin) {
    let tailPart = `/api/conn_manage/online_client/?Authorization=${localStorage.getItem("token")}`;

    let basePath = window.location.pathname.replace("/app/", "");
    if (import.meta.env.VITE_ROUTE_MODE === "WebHistory") {
      if (import.meta.env.VITE_WEB_BASE_DIR) {
        basePath = `${import.meta.env.VITE_WEB_BASE_DIR}`;
      } else {
        basePath = "";
      }
    }
    let sseUrl = `${basePath}${tailPart}`;
    let source = new EventSource(sseUrl);
    source.onmessage = function (event) {
      let onlineInfo = JSON.parse(event.data) as OnlineInfo;
      if (onlineInfo.code === 0) {
        data.is_admin = true;
        data.host_list = onlineInfo.data;
      }
    };
  }
}


onMounted(() => {
  getOnlineClient();
})
</script>


<style scoped></style>