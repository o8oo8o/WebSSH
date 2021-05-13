<template>
  <el-container>
    <el-header
      style="
        text-align: left;
        height: 28px;
        padding-left: 0px;
        padding-right: 0px;
      "
    >
      <el-row>
        <el-col :span="12">
          <el-button-group>
            <el-popover placement="bottom" trigger="click" :width="600">
              <template #reference>
                <el-button type="primary" icon="el-icon-menu">打开</el-button>
              </template>
              <el-table :data="host_list" height="260">
                <el-table-column
                  fixed="left"
                  width="150"
                  property="name"
                  label="名称"
                ></el-table-column>
                <el-table-column
                  width="150"
                  property="address"
                  label="主机"
                ></el-table-column>
                <el-table-column
                  width="100"
                  property="user"
                  label="用户"
                ></el-table-column>
                <el-table-column
                  width="60"
                  property="port"
                  label="端口"
                ></el-table-column>
                <el-table-column label="操作" fixed="right" width="300">
                  <template #default="scope">
                    <el-button size="mini" @click="editHost(scope.row)"
                      >编辑</el-button
                    >

                    <el-popconfirm
                      confirmButtonText="删除"
                      cancelButtonText="取消"
                      icon="el-icon-info"
                      iconColor="red"
                      title="确定删除吗"
                      @confirm="deleteHost(scope.row)"
                    >
                      <template #reference>
                        <el-button size="mini" type="danger">删除</el-button>
                      </template>
                    </el-popconfirm>

                    <el-button
                      size="mini"
                      type="primary"
                      @click="connectHost(scope.row)"
                      >连接</el-button
                    >
                  </template>
                </el-table-column>
              </el-table>
            </el-popover>

            <el-button
              type="primary"
              @click="newHost"
              icon="el-icon-circle-plus"
              >新建</el-button
            >

            <el-button type="primary" @click="toManage" icon="el-icon-s-custom"
              >管理</el-button
            >

            <el-dialog
              :title="mode == 0 ? '新增主机' : '更新主机'"
              v-model="host_dialog_visible"
              width="60%"
            >
              <el-form label-width="80px" ref="host_from" size="small">
                <el-form-item label="名称" prop="name">
                  <el-input
                    v-model.trim="name"
                    minlength="1"
                    maxlength="30"
                    show-word-limit
                    placeholder="请输入名称"
                  ></el-input>
                </el-form-item>
                <el-form-item label="主机" prop="address">
                  <el-input
                    v-model.trim="address"
                    minlength="1"
                    maxlength="60"
                    show-word-limit
                    placeholder="请输入主机地址"
                  ></el-input>
                </el-form-item>
                <el-form-item label="用户" prop="user">
                  <el-input
                    minlength="1"
                    maxlength="60"
                    v-model.trim="user"
                    show-word-limit
                    placeholder="请输入用户名"
                  ></el-input>
                </el-form-item>
                <el-form-item label="密码" prop="pwd">
                  <el-input
                    minlength="1"
                    maxlength="60"
                    v-model.trim="pwd"
                    type="passrowd"
                    show-password
                    show-word-limit
                    placeholder="密码"
                  ></el-input>
                </el-form-item>

                <el-row>
                  <el-col :span="9">
                    <el-form-item label="端口" prop="port">
                      <el-input-number
                        v-model="port"
                        :min="1"
                        :max="65535"
                        label="端口"
                      ></el-input-number>
                    </el-form-item>
                  </el-col>

                  <el-col :span="5">
                    <el-form-item label="字体颜色">
                      <el-color-picker v-model="foreground"></el-color-picker>
                    </el-form-item>
                  </el-col>

                  <el-col :span="5">
                    <el-form-item label="背景颜色">
                      <el-color-picker v-model="background"></el-color-picker>
                    </el-form-item>
                  </el-col>

                  <el-col :span="5">
                    <el-form-item label="光标颜色">
                      <el-color-picker v-model="cursor_color"></el-color-picker>
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-row>
                  <el-col :span="9">
                    <el-form-item label="字体">
                      <el-select
                        style="width: 130px"
                        v-model="font_family"
                        placeholder="请选择字体"
                      >
                        <el-option label="Courier" value="Courier" />
                        <el-option label="Courier New" value="Courier New" />
                        <el-option label="Menlo" value="Menlo" />
                        <el-option label="Monaco" value="Monaco" />
                        <el-option label="monospace" value="monospace" />
                      </el-select>
                    </el-form-item>
                  </el-col>

                  <el-col :span="5">
                    <el-form-item label="字体大小">
                      <el-select
                        v-model="font_size"
                        placeholder="请选择字体大小"
                      >
                        <el-option label="8" value="8" />
                        <el-option label="12" value="12" />
                        <el-option label="14" value="14" />
                        <el-option label="16" value="16" />
                        <el-option label="18" value="18" />
                        <el-option label="20" value="20" />
                        <el-option label="22" value="22" />
                        <el-option label="24" value="24" />
                        <el-option label="26" value="26" />
                        <el-option label="28" value="28" />
                      </el-select>
                    </el-form-item>
                  </el-col>

                  <el-col :span="5">
                    <el-form-item label="光标样式">
                      <el-select
                        v-model="cursor_style"
                        placeholder="请选择光标样式"
                      >
                        <el-option label="块状" value="block" />
                        <el-option label="下划线" value="underline" />
                        <el-option label="竖线" value="bar" />
                      </el-select>
                    </el-form-item>
                  </el-col>

                  <el-col :span="5">
                    <el-form-item label="Shell">
                      <el-select v-model="shell" placeholder="请选择Shell">
                        <el-option label="bash" value="bash" />
                        <el-option label="csh" value="csh" />
                        <el-option label="zsh" value="zsh" />
                      </el-select>
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-form>
              <template #footer>
                <span class="dialog-footer">
                  <el-button @click="host_dialog_visible = false"
                    >取消</el-button
                  >
                  <el-button type="success" @click="connect">连接</el-button>
                </span>
                &nbsp;&nbsp;&nbsp;&nbsp;

                <span v-if="mode == 0" class="dialog-footer">
                  <el-button type="primary" @click="createHost(false)"
                    >保存</el-button
                  >
                  <el-button type="primary" @click="createHost(true)"
                    >连接并保存</el-button
                  >
                </span>
                <span v-if="mode == 1" class="dialog-footer">
                  <el-button type="primary" @click="updateHost(false)"
                    >更新</el-button
                  >
                  <el-button type="primary" @click="updateHost(true)"
                    >连接并更新</el-button
                  >
                </span>
              </template>
            </el-dialog>
            <el-dialog
              v-model="file_dialog_visible"
              width="80%"
              custom-class="huang"
            >
              <template #title>
                <span v-html="title"></span>
              </template>
              <el-button-group>
                <el-button
                  v-for="(item, index) in dir_info.paths"
                  :key="index"
                  @click="listDir(item.dir)"
                  >{{ item.name }}</el-button
                >
              </el-button-group>

              <el-table :data="dir_info.files" height="400">
                <el-table-column
                  prop="name"
                  label="文件名"
                  fixed="left"
                  sortable
                >
                  <template #default="scope">
                    <el-button
                      v-if="scope.row.type === 'f'"
                      @click="downloadFile(scope.row)"
                      type="text"
                      size="small"
                      icon="el-icon-tickets"
                      style="color: green"
                      >{{ scope.row.name }}
                    </el-button>
                    <el-button
                      v-if="scope.row.type === 'd'"
                      @click="listDir(scope.row.path)"
                      type="text"
                      size="small"
                      icon="el-icon-folder-opened"
                      >{{ scope.row.name }}
                    </el-button>
                  </template>
                </el-table-column>
                <el-table-column prop="size" label="大小" width="100" sortable>
                </el-table-column>
                <el-table-column
                  prop="mode"
                  label="权限"
                  width="100"
                  sortable
                ></el-table-column>
                <el-table-column
                  prop="mod_time"
                  label="修改日期"
                  width="180"
                  sortable
                ></el-table-column>
                <el-table-column label="操作" width="100" fixed="right">
                  <template #default="scope">
                    <el-button
                      v-if="scope.row.type == 'f'"
                      @click="downloadFile(scope.row)"
                      type="success"
                      icon="el-icon-download"
                      >下载</el-button
                    >
                    <el-button
                      v-else
                      type="primary"
                      icon="el-icon-upload2"
                      @click="uploadFile(scope.row)"
                      >上传</el-button
                    >
                  </template>
                </el-table-column>
              </el-table>
            </el-dialog>
          </el-button-group>
        </el-col>
        <el-col :span="6"> </el-col>
        <el-col :span="6" style="text-align: right">
          <el-button type="success" @click="toGitHub">GitHub</el-button>
        </el-col>
      </el-row>
    </el-header>
    <div>
      <el-tabs
        v-model="current_host.session_id"
        type="card"
        closable
        @tab-remove="removeTab"
        @tab-click="selectTab"
      >
        <el-tab-pane
          v-for="item in host_tabs"
          :key="item.session_id"
          :label="item.name"
          :name="item.session_id"
        >
          <template #label>
            <el-button-group>
              <el-popover placement="bottom" :width="400" trigger="hover">
                <template #reference>
                  <el-button
                    round
                    :type="
                      item.session_id === current_host.session_id
                        ? 'primary'
                        : 'info'
                    "
                    >{{ item.name }}</el-button
                  >
                </template>
                <div>
                  <div>
                    <el-input disabled v-model="item.session_id">
                      <template #prepend>会话</template>
                    </el-input>
                  </div>
                  <div>
                    <el-input disabled v-model="item.address">
                      <template #prepend>主机</template>
                    </el-input>
                  </div>
                  <div>
                    <el-input disabled v-model="item.user">
                      <template #prepend>用户</template>
                    </el-input>
                  </div>
                  <div>
                    <el-input disabled v-model="item.port">
                      <template #prepend>端口</template>
                    </el-input>
                  </div>
                </div>
              </el-popover>

              <el-tooltip
                class="item"
                effect="dark"
                content="文件传输"
                placement="top"
              >
                <el-button
                  round
                  :type="
                    item.session_id === current_host.session_id
                      ? 'primary'
                      : 'info'
                  "
                  @click="listDir('/', item)"
                  icon="el-icon-sort"
                ></el-button>
              </el-tooltip>
            </el-button-group>
          </template>
          <template #default>
            <div style="margin: 1px">
              <div :id="item.session_id"></div>
            </div>
          </template>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-container>
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
import { ElMessage, ElPopover } from "element-plus";
import { Router, useRoute, useRouter } from "vue-router";

let route: Router;

enum Mode {
  "create" = 0,
  "update" = 1,
}

interface Host {
  id: number;
  name: string;
  address: string;
  user: string;
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
  term: Terminal;
  fit: FitAddon;
}

interface VerifyFromData {
  host: Host;
  is_success: boolean;
}

interface Path {
  dir: string;
  name: string;
}

interface File {
  name: string;
  mod_time: string;
  mode: string;
  path: string;
  type: "d" | "f";
  size: number;
}

interface DirInfo {
  current_dir: string;
  dir_count: number;
  file_count: number;
  files: Array<File>;
  paths: Array<Path>;
}

let data = reactive({
  mode: Mode.create,
  id: 0,
  name: "",
  address: "",
  user: "",
  pwd: "",
  port: 22,
  h: 20,
  w: 80,
  session_id: "",
  background: "#000000",
  foreground: "#FFFFFF",
  cursor_color: "#FFFFFF",
  font_family: "Courier",
  font_size: 16,
  cursor_style: "block",
  shell: "bash",

  upload_path: "",
  download_path: "",
  host_list: [] as Array<Host>,
  host_tabs: [] as Array<Host>,

  current_host: { session_id: "" } as Host,
  host_dialog_visible: false,
  file_dialog_visible: false,
  dir_info: {} as DirInfo,
});

const title = computed(() => {
  let titleHtml = `<span style="color:red;">当前名称:${data.current_host.name} &nbsp;&nbsp;&nbsp;当前主机:${data.current_host.address}</span>`;
  return titleHtml;
});

/**
 * 验证输入的主机信息
 */
function verifyFrom(): VerifyFromData {
  let verifyFromData: VerifyFromData = {
    host: {} as Host,
    is_success: false,
  };

  if (data.name.length === 0) {
    ElMessage.error("名称不能为空");
    return verifyFromData;
  }

  if (data.name.length > 30) {
    ElMessage.error("名称不能大于30个字符");
    return verifyFromData;
  }

  if (data.address.length === 0) {
    ElMessage.error("主机不能为空");
    return verifyFromData;
  }

  if (data.address.length > 60) {
    ElMessage.error("主机不能大于60个字符");
    return verifyFromData;
  }

  if (data.user.length === 0) {
    ElMessage.error("用户名不能为空");
    return verifyFromData;
  }

  if (data.user.length > 60) {
    ElMessage.error("用户名不能大于60个字符");
    return verifyFromData;
  }

  if (data.user.length === 0) {
    ElMessage.error("用户名不能为空");
    return verifyFromData;
  }

  if (data.user.length > 60) {
    ElMessage.error("用户名不能大于60个字符");
    return verifyFromData;
  }

  if (data.pwd.length === 0) {
    ElMessage.error("密码不能为空");
    return verifyFromData;
  }

  if (data.user.length > 60) {
    ElMessage.error("密码不能大于60个字符");
    return verifyFromData;
  }

  if (!data.port) {
    ElMessage.error("端口输入错误,必须是1-65535");
    return verifyFromData;
  }

  if (data.port < 1 || data.port > 65535) {
    ElMessage.error("端口范围错误,必须是1-65535");
    return verifyFromData;
  }
  let h = {
    id: data.id,
    name: data.name,
    address: data.address,
    user: data.user,
    pwd: data.pwd,
    port: data.port,
    session_id: data.session_id,
    background: data.background,
    foreground: data.foreground,
    cursor_color: data.cursor_color,
    font_family: data.font_family,
    font_size: data.font_size,
    cursor_style: data.cursor_style,
    shell: data.shell,
  };
  let result: VerifyFromData = {
    host: h as Host,
    is_success: true,
  };
  return result;
}

/**
 * 清空表单数据
 */

function cleanFrom() {
  data.id = 0;
  data.name = "";
  data.address = "";
  data.user = "";
  data.pwd = "";
  data.port = 22;
  data.session_id = "";
  data.background = "#000000";
  data.foreground = "#FFFFFF";
  data.cursor_color = "#FFFFFF";
  data.font_family = "Courier";
  data.font_size = 16;
  data.cursor_style = "block";
  data.shell = "bash";
}

/**
 * 连接
 */
function connect() {
  let result = verifyFrom();
  if (!result.is_success) {
    return;
  }
  connectHost(result.host);
}

/**
 * 打开文件列表
 */
function listDir(dir: string, h: Host) {
  data.file_dialog_visible = true;

  if (h) {
    setCurrentAcitveHost(h.session_id);
  }
  let host = { ...data.current_host };

  if (!host.hasOwnProperty("session_id")) {
    // 没有连接主机
    return;
  }

  let xhttp = new Xhttp();
  let url = `/api/file?session_id=${host.session_id}&path=${dir}`;
  xhttp
    .get(url)
    .then((res) => res.json())
    .then((json) => {
      if (json.code === 0) {
        data.dir_info = json.data;
      }
    });
}

/**
 * 上传文件
 */
function uploadFile(file: File) {
  function upload(fileList: FileList) {
    let formData = new FormData();
    formData.append("session_id", data.current_host.session_id);
    formData.append("path", file.path);
    for (let i = 0; i < fileList.length; i++) {
      formData.append("file", fileList[i]);
    }
    let xhttp = new Xhttp();
    xhttp
      .put("/api/file", { body: formData })
      .then((res) => res.json())
      .then((json) => {
        if (json.code === 0) {
          ElMessage.success(json.msg);
        }
      });
  }

  let fileInput = document.createElement("input");
  fileInput.type = "file";
  fileInput.multiple = true;

  fileInput.onchange = function (f: any) {
    let fileList = fileInput.files as FileList;
    upload(fileList);
  };
  fileInput.click();
}

/**
 * 下载文件(只能是文件,不能是目录)
 */
function downloadFile(file: File) {
  let formData = new FormData();
  formData.append("session_id", data.current_host.session_id);
  formData.append("path", file.path);
  let xhttp = new Xhttp();
  xhttp.post("/api/file", { body: formData }).then((res) =>
    res.blob().then((blob) => {
      var a = document.createElement("a");
      var url = window.URL.createObjectURL(blob);
      a.href = url;
      a.download = file.name;
      a.click();
      window.URL.revokeObjectURL(url);
    })
  );
}

/**
 * 获取所有主机列表
 */
function getallHost() {
  let xhttp = new Xhttp();
  xhttp.get("/api/host").then((response) =>
    response.json().then((json) => {
      if (json.code === 0) {
        if (json.data != null) {
          data.host_list = json.data;
        }
      }
    })
  );
}

/**
 * 创建或更新主机
 */
function createOrUpdateHost(host: Host, m: Mode) {
  // 关闭模态框,from 表单验证后续在搞 :rules="host_from_rules"
  let xhttp = new Xhttp();
  if (m == 0) {
    for (let i = 0; i < data.host_list.length; i++) {
      // 数据库中name是unique约束
      let item = data.host_list[i];
      if (item.name == host.name) {
        ElMessage.error("名称已经存在,请修改");
        return;
      }
    }
  }

  // 关闭模态框
  data.host_dialog_visible = false;
  let fm = new FormData();
  fm.append("id", String(host.id));
  fm.append("name", host.name);
  fm.append("address", host.address);
  fm.append("user", host.user);
  fm.append("pwd", host.pwd);
  fm.append("port", String(host.port));
  fm.append("font_size", String(host.font_size));
  fm.append("background", host.background);
  fm.append("foreground", host.foreground);
  fm.append("cursor_color", host.cursor_color);
  fm.append("font_family", host.font_family);
  fm.append("cursor_style", host.cursor_style);
  fm.append("shell", host.shell);

  if (m == 0) {
    xhttp
      .post("/api/host", { body: fm })
      .then((response) => response.json())
      .then((json) => {
        if (json.code === 0) {
          data.host_list = json.data;
          cleanFrom();
        }
      });
  } else {
    xhttp
      .put("/api/host", { body: fm })
      .then((response) => response.json())
      .then((json) => {
        if (json.code === 0) {
          data.host_list = json.data;
          cleanFrom();
        }
      });
  }
}

/**
 * 进入主机创建模式
 */
function newHost() {
  cleanFrom();
  data.host_dialog_visible = true;
  data.mode = 0;
}

/**
 * 创建主机并保存(也可以创建主机并保存且保存)
 */
function createHost(isConnect: boolean = false) {
  // 创建模式
  data.mode = Mode.create;

  let result = verifyFrom();
  if (!result.is_success) {
    return;
  }
  createOrUpdateHost(result.host, Mode.create);
  if (isConnect) {
    connectHost(result.host);
  }
}

/**
 * 编辑主机
 */
function editHost(row: Host) {
  // 打开模态框
  data.host_dialog_visible = true;

  // 编辑模式
  data.mode = Mode.update;
  data.id = row.id;
  data.address = row.address;
  data.name = row.name;
  data.user = row.user;
  data.pwd = row.pwd;
  data.port = row.port;
  data.background = row.background;
  data.foreground = row.foreground;
  data.cursor_color = row.cursor_color;
  data.font_family = row.font_family;
  data.font_size = row.font_size;
  data.cursor_style = row.cursor_style;
  data.shell = row.shell;
}

/**
 * 更新主机信息
 */
function updateHost(isConnect: boolean = false) {
  let result = verifyFrom();
  if (!result.is_success) {
    return;
  }
  createOrUpdateHost(result.host, Mode.update);
  if (isConnect) {
    connectHost(result.host);
  }
}

/**
 * 删除已经保存的主机
 */
function deleteHost(row: any) {
  let fromData = new FormData();
  fromData.append("id", row.id);
  let xhttp = new Xhttp();
  xhttp
    .delete("/api/host", { body: fromData })
    .then((response) => response.json())
    .then((json) => {
      if (json.code === 0) {
        if (json.data != null) {
          data.host_list = json.data;
          return;
        }
        data.host_list = [];
      }
    });
}

/**
 * 获取会话session_id
 */
function getSessionId(host: Host): Promise<string> {
  let fm = new FormData();
  fm.append("id", String(host.id));
  fm.append("name", host.name);
  fm.append("address", host.address);
  fm.append("user", host.user);
  fm.append("pwd", host.pwd);
  fm.append("port", String(host.port));
  fm.append("shell", host.shell);

  let xhttp = new Xhttp();
  return xhttp
    .patch("/api/host", { body: fm })
    .then((response) => response.json())
    .then((json) => {
      if (json.code === 0) {
        return json.data;
      }
    });
}

/**
 * 连接已经保存过的主机
 */
function connectHost(host: Host) {
  // 关闭模态框
  data.host_dialog_visible = false;

  getSessionId(host).then((sessionId) => {
    // 必须要解开,否则出现 Avoid app logic that relies on enumerating keys on a component instance. The keys will be empty in production mode to avoid performance overhead
    let connHost = { ...host };
    connHost.session_id = sessionId;

    // 窗口大小适应插件
    connHost.fit = new FitAddon();

    connHost.term = new Terminal({
      cursorBlink: true,
      theme: {
        background: connHost.background,
        foreground: connHost.foreground,
        cursor: connHost.cursor_color,
      },
      fontSize: connHost.font_size,
      fontFamily: connHost.font_family,
      cursorStyle: connHost.cursor_style,
    });

    // 加载窗口大小自适应插件
    connHost.term.loadAddon(connHost.fit);

    // 添加tab 页面
    data.host_tabs.push(connHost);

    // 设置当前激活的host
    data.current_host = { ...connHost };

    nextTick(() => {
      connHost.term.open(
        document.getElementById(connHost.session_id) as HTMLElement
      );

      (document.getElementById(
        connHost.session_id
      ) as HTMLElement).style.height =
        Math.floor(window.innerHeight - 70) + "px";
      connHost.fit.fit();

      let param = `h=${connHost.term.rows}&w=${connHost.term.cols}&session_id=${connHost.session_id}`;
      let sock_url = `${location.protocol == "http:" ? "ws://" : "wss://"}${
        location.host
      }/api/ssh?${param}`;

      connHost.term.loadAddon(new AttachAddon(new WebSocket(sock_url)));

      // 清空 from 表单数据
      cleanFrom();
    });
  });
}

/**
 * 删除tab
 */
function removeTab(tabId: string) {
  let removeIndex = 0;
  data.host_tabs.forEach((host, index) => {
    if (host.session_id === tabId) {
      removeIndex = index;
    }
  });

  // 销毁term 对象
  data.host_tabs[removeIndex].fit.dispose();
  data.host_tabs[removeIndex].term.dispose();

  // 从tab页签中删除
  data.host_tabs.splice(removeIndex, 1);

  // 如果没有打开的tab页签,就直接返回
  if (data.host_tabs.length === 0) {
    return;
  }

  // 如果打开的tab页签只有一个,就把这个tab页签设置成激活状态
  if (data.host_tabs.length === 1) {
    let activeHost = { ...data.host_tabs[0] };
    setCurrentAcitveHost(activeHost.session_id);
    return;
  }

  // 如果打开的tab页签只有一个以上,删除以后把下一个tab页签设置成激活
  if (data.host_tabs.length > 1) {
    let activeHost = { ...data.host_tabs[removeIndex] };
    setCurrentAcitveHost(activeHost.session_id);
  }
}

/***
 * 点击切换tab
 */
function selectTab(tab: any) {
  let sessionId = tab.props.name;
  setCurrentAcitveHost(sessionId);
}

/**
 * 设置当前正在使用的主机
 */
function setCurrentAcitveHost(sessionId: string) {
  data.host_tabs.forEach((host: Host) => {
    if (host.session_id === sessionId) {
      data.current_host = { ...host };
      return;
    }
  });
  windowResize();
}

/**
 * 更改窗口大小
 */
function windowResize() {
  let currentHost = data.current_host;

  if (currentHost.session_id === "") {
    // console.log("还没有连接主机");
    return;
  }

  // 没有在主机连接路由页面
  if (route.currentRoute.value.name !== "Home") {
    return;
  }

  nextTick(() => {
    (document.getElementById(
      currentHost.session_id
    ) as HTMLElement).style.height = Math.floor(window.innerHeight - 70) + "px";

    currentHost.fit.fit();
    if (data.h !== currentHost.term.rows || data.w !== currentHost.term.cols) {
      let url = `/api/ssh?w=${currentHost.term.cols}&h=${currentHost.term.rows}&session_id=${currentHost.session_id}`;
      let xhttp = new Xhttp();
      xhttp.patch(url);
    }
    /* else {
      console.log(`窗口大小没有变化`);
    } */
    data.h = Math.floor(currentHost.term.rows);
    data.w = Math.floor(currentHost.term.cols);
  });
}

// 报告连接状态
function reportConnectStatus() {
  setInterval(function () {
    let fm = new FormData();
    let sessionIdList = new Array<string>();
    data.host_tabs.forEach((hont) => {
      fm.append("ids", hont.session_id);
    });

    let xhttp = new Xhttp();
    xhttp
      .post("/api/status", { body: fm })
      .then((response) => response.json())
      .then((json) => {
        if (json.code === 0) {
          // console.log(json);
        }
      });
  }, 10000);
}

// 跳转到管理页面
function toManage() {
  window.open("/#/manage", "_blank");
}

// 跳转到管理页面
function toGitHub() {
  window.open("https://github.com/o8oo8o/GoWebSSH", "_blank");
}

export default defineComponent({
  name: "terminal",

  setup(props: any, context) {
    onMounted(() => {
      route = useRouter();
      reportConnectStatus();
      getallHost();
      window.onresize = windowResize;
      window.onbeforeunload = function () {
        return "关闭吗";
      };
    });
    return {
      ...toRefs(data),
      title,
      connect,
      newHost,
      editHost,
      listDir,
      uploadFile,
      downloadFile,
      createHost,
      updateHost,
      deleteHost,
      connectHost,
      removeTab,
      selectTab,
      toManage,
      toGitHub,
    };
  },
});
</script>


<style scoped>
.huang {
  margin-top: 0px;
}
</style>

