<template>
  <el-container>
    <el-header style="
        height: 32px;
        padding-left: 0px;
        padding-right: 0px;
        background-color: #409eff;
      ">
      <el-row>
        <el-col :span="12">
          <el-button-group>
            <!-- 打开已存在主机配置 -->
            <el-popover placement="bottom" trigger="click" :width="800">
              <template #reference>
                <el-button type="primary" :icon="Menu">打开</el-button>
              </template>
              <el-table :data="filterHostTable" height="360" :show-overflow-tooltip="true">
                <el-table-column sortable fixed="left" width="150" property="name" label="名称"></el-table-column>
                <el-table-column sortable width="150" property="address" label="主机"></el-table-column>
                <el-table-column sortable width="100" property="user" label="用户"></el-table-column>
                <el-table-column sortable width="90" property="port" label="端口"></el-table-column>
                <el-table-column label="操作" fixed="right">
                  <template #header>
                    <el-input v-model="searchHost" placeholder="名称及主机搜索" />
                  </template>
                  <template #default="scope">
                    <el-button @click="editHost(scope.row)">编辑</el-button>
                    <el-popconfirm confirmButtonText="删除" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                      title="确定删除吗" @confirm="deleteHost(scope.row)">
                      <template #reference>
                        <el-button type="danger">删除</el-button>
                      </template>
                    </el-popconfirm>
                    <el-button type="primary" @click="connectHost(scope.row)">连接</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-popover>

            <el-button type="primary" @click="newHost" :icon="CirclePlus">新建</el-button>
            <!-- 执行命令及收藏 -->
            <el-popover placement="bottom" trigger="click" :width="800">
              <template #reference>
                <el-button type="primary" :icon="Edit">执行命令</el-button>
              </template>
              <el-form :model="cmd">
                <el-form-item label="执行命令">
                  <el-input v-model="cmd.data" type="textarea" autocomplete="off" placeholder="命令或脚本" />
                </el-form-item>
                <el-row>
                  <el-col :span="10">
                    <el-form-item label="会话选择">
                      <el-radio-group v-model="cmd.node">
                        <el-radio value="current">当前会话</el-radio>
                        <el-radio value="all">所有会话</el-radio>
                      </el-radio-group>
                    </el-form-item>
                  </el-col>
                  <el-col :span="14">
                    <el-form-item>
                      <el-input v-model="cmd.name" maxlength="32" show-word-limit placeholder="如果需要收藏命令,请输入名称">
                        <template #append>
                          <el-button-group style="color:blue">
                            <el-button @click="addCmdNote">收藏</el-button>
                            <el-button @click="execCmd">执行</el-button>
                          </el-button-group>
                        </template>
                      </el-input>
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-form>
            </el-popover>

            <!-- 命令收藏列表 -->
            <el-popover placement="bottom" trigger="click" :width="800">
              <template #reference>
                <el-button type="primary" :icon="Star">命令收藏</el-button>
              </template>
              <el-table :data="filterCmdNoteTable" height="260">
                <el-table-column sortable width="180" :show-overflow-tooltip="true" property="cmd_name"
                  label="名称"></el-table-column>
                <el-table-column sortable property="cmd_data" label="命令">
                  <template #default="scope">
                    <el-popover effect="light" trigger="hover" placement="right" width="auto">
                      <template #default>
                        <div>命令详情</div>
                        <div>
                          <el-input v-model="scope.row.cmd_data" style="width: 600px"
                            :autosize="{ minRows: 4, maxRows: 20 }" type="textarea" :disabled="true" />
                        </div>
                        <div>
                          </br>
                          <el-button-group>
                            <el-tooltip effect="dark" content="执行命令,发送到所有会话" placement="top-start">
                              <el-button type="warning" @click="execCmdAllSession(scope.row)">发送所有会话</el-button>
                            </el-tooltip>
                            <el-tooltip effect="dark" content="执行命令,发送到当前会话" placement="top-start">
                              <el-button type="primary" @click="execCmdCurrentSession(scope.row)">发送当前会话</el-button>
                            </el-tooltip>
                          </el-button-group>
                        </div>
                      </template>
                      <template #reference>
                        {{ scope.row.cmd_data.substring(0, 15) + "..." }}
                      </template>
                    </el-popover>
                  </template>
                </el-table-column>
                <el-table-column label="操作" fixed="right" width="320">
                  <template #header>
                    <el-input v-model="searchCmdNote" placeholder="名称搜索" />
                  </template>
                  <template #default="scope">
                    <el-button-group>
                      <el-popconfirm confirmButtonText="删除" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                        title="确定删除吗" @confirm="delCmdNote(scope.row.id)">
                        <template #reference>
                          <el-button type="danger">删除</el-button>
                        </template>
                      </el-popconfirm>
                      <el-tooltip effect="dark" content="执行命令,发送到所有会话" placement="top-start">
                        <el-button type="warning" @click="execCmdAllSession(scope.row)">发送所有会话</el-button>
                      </el-tooltip>
                      <el-tooltip effect="dark" content="执行命令,发送到当前会话" placement="top-start">
                        <el-button type="primary" @click="execCmdCurrentSession(scope.row)">发送当前会话</el-button>
                      </el-tooltip>
                    </el-button-group>
                  </template>
                </el-table-column>
              </el-table>
            </el-popover>
          </el-button-group>
        </el-col>
        <el-col :span="12" style="text-align: right">
          <el-button-group>
            <el-popover placement="top-start" title="详情" :width="300" trigger="hover">
              <template #reference>
                <el-button type="primary" :icon="User">{{ globalStore.userName }}</el-button>
              </template>
              <p><el-text type="info">用户名称:&nbsp;&nbsp;{{ globalStore.userDesc }}</el-text></p>
              <p><el-text type="info">过期时间:&nbsp;&nbsp;{{ globalStore.userExpiryAt }}</el-text></p>
            </el-popover>

            <el-button type="primary" :icon="Setting" @click="data.modify_pwd_dialog_visible = true">修改密码</el-button>

            <!-- admin 角色才能管理 -->
            <el-button v-if="globalStore.isAdmin === 'Y'" type="danger" :icon="Coin" @click="toManage">管理</el-button>
            <!-- <el-popconfirm v-if="globalStore.isAdmin === 'Y'" confirmButtonText="确定" cancelButtonText="取消"
              icon="el-icon-info" iconColor="red" title="确定离开此页面吗" @confirm="toManage">
              <template #reference>
                <el-button type="danger" :icon="Coin">管理</el-button>
              </template>
            </el-popconfirm> -->

            <el-popconfirm confirmButtonText="退出" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
              title="确定退出吗" @confirm="logout">
              <template #reference>
                <el-button :icon="CircleClose" type="danger">退出</el-button>
              </template>
            </el-popconfirm>
          </el-button-group>
        </el-col>
      </el-row>
    </el-header>

    <div>
      <!-- 修改密码 -->
      <el-dialog v-model="data.modify_pwd_dialog_visible" title="修改密码" width="500" center>
        <el-form>
          <el-form-item>
            <el-input v-model="data.new_pwd_one" trim type="password" minlength="3" maxlength="64" show-word-limit
              show-password clearable placeholder="输入新密码">
              <template #prepend>输入新密码</template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-input v-model="data.new_pwd_two" trim type="password" minlength="3" maxlength="64" show-word-limit
              show-password clearable placeholder="确认新密码">
              <template #prepend>确认新密码</template>
            </el-input>
          </el-form-item>
        </el-form>
        <template #footer>
          <div class="dialog-footer">
            <el-button @click="data.modify_pwd_dialog_visible = false">取消</el-button>
            <el-button type="primary" @click="modifyPassword">
              提交
            </el-button>
          </div>
        </template>
      </el-dialog>

      <!-- SSH主机配置弹窗 -->
      <el-dialog :title="data.mode == 0 ? '新增主机' : '更新主机'" v-model="data.host_dialog_visible" width="80%" top="60px">
        <el-form label-width="80px" ref="host_from">
          <el-collapse v-model="data.host_config_collapse">
            <el-collapse-item title="基础配置" name="1">
              <el-row>
                <el-col :span="16">
                  <el-form-item label="名称" prop="name">
                    <el-input v-model.trim="data.name" minlength="1" maxlength="30" show-word-limit
                      placeholder="请输入名称"></el-input>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="16">
                  <el-form-item label="主机" prop="address">
                    <el-input v-model.trim="data.address" minlength="1" maxlength="60" show-word-limit
                      placeholder="请输入主机地址"></el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="网络" prop="net_type">
                    <el-radio-group v-model="data.net_type">
                      <el-radio value="tcp4">IPv4</el-radio>
                      <el-radio value="tcp6">IPv6</el-radio>
                    </el-radio-group>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="16">
                  <el-form-item label="用户" prop="user">
                    <el-input minlength="1" maxlength="60" v-model.trim="data.user" show-word-limit
                      placeholder="请输入用户名"></el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="端口" prop="port">
                    <el-input-number v-model="data.port" :min="1" :max="65535"></el-input-number>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-form-item label="认证方式">
                  <el-radio-group v-model="data.auth_type">
                    <el-radio value="pwd">密码</el-radio>
                    <el-radio value="cert">密钥</el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-row>
              <el-row v-if="data.auth_type === 'cert'">
                <el-col :span="16">
                  <el-form-item label="密钥">
                    <el-input v-model="data.cert_data" type="textarea" placeholder="请输入密钥内容或上传" />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="上传">
                    <el-button type="primary" @click="addCertFile">上传密钥文件</el-button>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="16">
                  <el-form-item v-if="data.auth_type === 'cert'" label="密钥口令" prop="cert_pwd">
                    <el-input minlength="" maxlength="60" v-model.trim="data.cert_pwd" type="passrowd" show-password
                      show-word-limit placeholder="密钥口令"></el-input>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="16">
                  <el-form-item v-if="data.auth_type === 'pwd'" label="SSH密码" prop="pwd">
                    <el-input minlength="1" maxlength="60" v-model.trim="data.pwd" type="passrowd" show-password
                      show-word-limit placeholder="SSH密码"></el-input>
                  </el-form-item>
                </el-col>
              </el-row>
            </el-collapse-item>

            <el-collapse-item title="高级配置" name="2">
              <el-row>
                <el-col :span="9">
                  <el-form-item label="终端类型" prop="pty_type">
                    <el-select style="width: 130px" v-model="data.pty_type" placeholder="请选择终端类型">
                      <el-option label="xterm-256color" value="xterm-256color" />
                      <el-option label="linux" value="linux" />
                      <el-option label="xtrem" value="xtrem" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="5">
                  <el-form-item label="字体颜色" prop="foreground">
                    <el-color-picker v-model="data.foreground"></el-color-picker>
                  </el-form-item>
                </el-col>
                <el-col :span="5">
                  <el-form-item label="背景颜色" prop="background">
                    <el-color-picker v-model="data.background"></el-color-picker>
                  </el-form-item>
                </el-col>
                <el-col :span="5">
                  <el-form-item label="光标颜色" prop="cursor_color">
                    <el-color-picker v-model="data.cursor_color"></el-color-picker>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="9">
                  <el-form-item label="字体">
                    <el-select style="width: 130px" v-model="data.font_family" placeholder="请选择字体">
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
                    <el-select v-model.number="data.font_size" placeholder="请选择字体大小">
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
                      <el-option label="30" value="30" />
                      <el-option label="32" value="32" />
                      <el-option label="34" value="34" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="4">
                  <el-form-item label="光标样式">
                    <el-select v-model="data.cursor_style" placeholder="请选择光标样式">
                      <el-option label="块状" value="block" />
                      <el-option label="下划线" value="underline" />
                      <el-option label="竖线" value="bar" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <!--
                        <el-form-item label="Shell">
                          <el-select v-model="data.shell" placeholder="请选择Shell">
                            <el-option label="sh" value="sh" />
                            <el-option label="bash" value="bash" />
                            <el-option label="csh" value="csh" />
                            <el-option label="zsh" value="zsh" />
                            <el-option label="(windows)powershell" value="powershell" />
                          </el-select>
                        </el-form-item>                        
                        -->
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="24">
                  <el-form-item label="连接命令">
                    <el-input v-model="data.init_cmd" type="textarea" :row="1" placeholder="请输入连接后执行命令" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row>
                <el-col :span="24">
                  <el-form-item label="连接横幅">
                    <el-input v-model="data.init_banner" type="textarea" :row="1" placeholder="请输入连接后提示横幅" />
                  </el-form-item>
                </el-col>
              </el-row>
            </el-collapse-item>
          </el-collapse>
        </el-form>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="data.host_dialog_visible = false">取消</el-button>
            <el-button type="success" @click="connect">连接</el-button>
          </span>
          &nbsp;&nbsp;&nbsp;&nbsp;
          <span v-if="data.mode == 0" class="dialog-footer">
            <el-button type="primary" @click="createHost(false)">保存</el-button>
            <el-button type="primary" @click="createHost(true)">连接并保存</el-button>
          </span>
          <span v-if="data.mode == 1" class="dialog-footer">
            <el-button type="primary" @click="updateHost(false)">更新</el-button>
            <el-button type="primary" @click="updateHost(true)">连接并更新</el-button>
          </span>
        </template>
      </el-dialog>

      <!-- SSH文件上传下载弹窗 -->
      <el-dialog v-model="data.file_dialog_visible" width="80%" custom-class="file-dialog" top="60px">
        <template #header>
          <span v-html="title"></span>
        </template>

        <el-button-group style="width:auto;display: flex; flex-wrap: nowrap;overflow-x: auto;">
          <el-button v-for="(item, index) in data.dir_info.paths" :key="index"
            @click="listDir(item.dir, data.current_host)">{{ item.name }}</el-button>
        </el-button-group>
        </br>

        <el-form-item style="margin-top: 10px;">
          <el-input v-model="data.sftp_current_dir" style="width: 100%;" placeholder="请输入路径" class="input-with-select">
            <template #append>
              <el-button-group style="color:blue">
                <el-button @click="listDir(data.sftp_current_dir, data.current_host)">进入</el-button>
                <el-button @click="uploadFile(data.sftp_current_dir)">上传</el-button>
                <el-button @click="createDir(data.sftp_current_dir, data.current_host)">创建目录</el-button>
                <el-button @click="listDir(data.sftp_current_dir, data.current_host)">刷新</el-button>
              </el-button-group>
            </template>
          </el-input>
        </el-form-item>
        </br>

        <el-row>
          <el-col :span="24">
            <el-progress :percentage="data.sftp_upload_percentage" />
          </el-col>
        </el-row>

        <el-table :data="data.dir_info.files" height="400" :show-overflow-tooltip="true">
          <el-table-column prop="name" label="文件名" fixed="left" sortable>
            <template #default="scope">
              <el-button v-if="scope.row.type === 'f'" @click="downloadFile(scope.row)" type="primary" link
                :icon="Files" style="color: green">{{ scope.row.name }}</el-button>
              <el-button v-if="scope.row.type === 'd'" @click="listDir(scope.row.path, data.current_host)"
                type="primary" link :icon="FolderOpened">{{ scope.row.name }}</el-button>
            </template>
          </el-table-column>
          <el-table-column prop="size" label="大小" width="100" sortable></el-table-column>
          <el-table-column prop="mode" label="权限" width="100" sortable></el-table-column>
          <el-table-column prop="mod_time" label="修改日期" width="180" sortable></el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="scope">
              <el-button-group>
                <el-button v-if="scope.row.type == 'f'" @click="downloadFile(scope.row)" type="success"
                  :icon="Bottom">下载</el-button>
                <el-button v-else type="primary" :icon="Upload" @click="uploadFile(scope.row.path)">上传</el-button>
                <el-popconfirm confirmButtonText="删除" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                  title="确定删除吗" @confirm="deleteFile(scope.row)">
                  <template #reference>
                    <el-button type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
      </el-dialog>

      <!-- 管理 -->
      <el-dialog title="系统管理" v-model="data.manage_dialog_visible" v-bind:fullscreen="true">
        <Manage></Manage>
      </el-dialog>
    </div>

    <div v-if="data.host_tabs.length === 0">
      <Empty></Empty>
    </div>
    <div v-else>
      <el-tabs v-model="data.current_host.session_id" type="card" closable @tab-remove="removeTab"
        @tab-click="selectTab">
        <el-tab-pane v-for="item in data.host_tabs" :key="item.session_id" :label="item.name" :name="item.session_id">
          <template #label>
            <el-button-group style="width:auto;display: flex; flex-wrap: nowrap;overflow-x: auto;">
              <el-popover placement="bottom" :width="400" trigger="hover">
                <template #reference>
                  <el-button :type="item.session_id === data.current_host.session_id
                    ? 'primary' : 'info'">
                    <span v-if="item.is_close" style="color:red">{{ item.name }}</span>
                    <span v-else="item.is_close" style="color:white">{{ item.name }}</span>
                  </el-button>
                </template>
                <div>
                  <div style="padding-top: 5px;">
                    <el-button-group>
                      <el-button type="primary" @click="connectHost(item, true)">重连</el-button>
                      <el-button type="primary" @click="item.term.clear()">清空缓冲区</el-button>
                    </el-button-group>
                  </div>
                  <div style="padding-top: 5px;">
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
                </div>
              </el-popover>

              <el-tooltip class="item" effect="dark" content="文件传输" placement="top">
                <el-button :type="item.session_id === data.current_host.session_id
                  ? 'primary'
                  : 'info'
                  " @click="listDir('/', item)" :icon="Sort"></el-button>
              </el-tooltip>
            </el-button-group>
          </template>
          <template #default>
            <div id="term-data" style="margin: 1px">
              <div :id="item.session_id" style="width: 100vw;height:100vh"></div>
            </div>
          </template>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-container>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, defineAsyncComponent } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElNotification, ElPopover } from "element-plus";
import { FolderOpened, Files, Bottom, Upload, Menu, CirclePlus, Coin, Sort, Edit, Setting, User, CircleClose, Star } from "@element-plus/icons-vue";
import axios, { type AxiosProgressEvent } from "axios";
import { useGlobalStore } from "@/stores/store";
import { Terminal } from "@xterm/xterm";
import { AttachAddon } from "@xterm/addon-attach";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import Empty from "./Empty.vue";
const Manage = defineAsyncComponent(() => import('./Manage.vue'))


let router = useRouter();
let globalStore = useGlobalStore();

enum Mode {
  "create" = 0,
  "update" = 1,
}

interface ResponseData {
  code: number;
  msg: string;
  data?: any;
}

/**
 * 连接Host对象
 */
interface Host {
  id: number;
  name: string;
  address: string;
  user: string;
  auth_type: "pwd" | "cert";
  net_type: "tcp4" | "tcp6";
  cert_data: string;
  cert_pwd: string;
  pwd: string;
  port: number;
  font_size: number;
  background: string;
  foreground: string;
  cursor_color: string;
  font_family: string;
  cursor_style: "block" | "underline" | "bar";
  shell: string;
  pty_type: "xterm-256color" | "xterm" | "linux";
  init_cmd: string;
  init_banner: string;
  session_id: string;
  term: Terminal;
  fit: FitAddon;
  ws: WebSocket;
  is_close: boolean;
}

/**
 * 表单验证
 */
interface VerifyFromData {
  host: Host;
  is_success: boolean;
}

/**
 * sftp Path
 */
interface Path {
  dir: string;
  name: string;
}

/**
 * sftp FileInfo
 */
interface FileInfo {
  name: string;
  mod_time: string;
  mode: string;
  path: string;
  type: "d" | "f";
  size: number;
}

/**
 * sftp DirInfo
 */
interface DirInfo {
  current_dir: string;
  dir_count: number;
  file_count: number;
  files: Array<FileInfo>;
  paths: Array<Path>;
}

let data = reactive({
  mode: Mode.create,
  id: 0,
  name: "",
  address: "",
  user: "",
  auth_type: "pwd",
  net_type: "tcp4",
  cert_data: "",
  cert_pwd: "",
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
  pty_type: "xterm-256color",
  init_cmd: "",
  init_banner: "",

  upload_path: "",
  download_path: "",
  host_list: [] as Array<Host>,
  host_tabs: [] as Array<Host>,

  current_host: { session_id: "" } as Host,
  host_config_collapse: ['1'],
  host_dialog_visible: false,
  file_dialog_visible: false,
  modify_pwd_dialog_visible: false,
  manage_dialog_visible: false,
  dir_info: {} as DirInfo,
  sftp_current_dir: "",
  sftp_upload_percentage: 0,
  new_pwd_one: "",
  new_pwd_two: "",
});

/**
 * 调试
 */
function debug() {
  console.log(data);
  console.log(data.current_host);
  console.log(data.host_list);
  console.log(data.host_tabs);
}

/**
 * 批量执行命令
 */
let cmd = reactive({ name: "", data: "", node: "current" });

interface CmdNode {
  id: number;
  cmd_name: string;
  cmd_data: string;
}

let cmdNotes = ref<Array<CmdNode>>([]);

/**
 * 搜索主机列表
 */
const searchHost = ref("");
const filterHostTable = computed(() =>
  data.host_list.filter(
    (i) =>
      !searchHost.value ||
      i.name.toLowerCase().includes(searchHost.value.toLowerCase()) ||
      i.address.toLowerCase().includes(searchHost.value.toLowerCase())
  )
)

/**
 * 搜索命令收藏列表
 */
const searchCmdNote = ref("");
const filterCmdNoteTable = computed(() =>
  cmdNotes.value.filter(
    (i) =>
      !searchCmdNote.value ||
      i.cmd_name.toLowerCase().includes(searchCmdNote.value.toLowerCase())
  )
)

/**
 * 状态报告定时器
 */
let statusSetInterval: number;

/**
 * sftp 文件传输弹窗title
 */
const title = computed(() => {
  let titleHtml = `<span style="color:red;">当前名称:${data.current_host.name} &nbsp;&nbsp;&nbsp;当前主机:${data.current_host.address}</span>`;
  return titleHtml;
});

/**
 * 修改密码
 */
function modifyPassword() {
  if (data.new_pwd_one.length < 2) {
    ElMessage.error("密码至少两个字符");
    return
  }
  if (data.new_pwd_two.length < 2) {
    ElMessage.error("密码至少两个字符");
    return
  }
  if (data.new_pwd_one !== data.new_pwd_two) {
    ElMessage.error("两次密码输入不一致");
    return
  }

  axios.patch<ResponseData>("/api/user/pwd", { "pwd": data.new_pwd_one }).then((ret) => {
    if (ret.data.code === 0) {
      ElMessage.success("密码修改成功");
    } else {
      ElMessage.error("密码修改失败");
    }
  }).catch(() => {
    ElMessage.error("密码修改错误");
  })
  data.modify_pwd_dialog_visible = false;
}

/**
 * 执行命令
 */
function execCmd() {
  if (cmd.node == "current") {
    execCmdCurrentSession({ "id": 0, "cmd_name": "", "cmd_data": cmd.data });
  }
  if (cmd.node == "all") {
    execCmdAllSession({ "id": 0, "cmd_name": "", "cmd_data": cmd.data });
  }
}

/**
 * 添加命令收藏
 */
function addCmdNote() {
  if (cmd.data.trim().length === 0) {
    ElMessage.error("收藏的命令不能为空");
    return;
  }

  if (cmd.name.trim().length === 0) {
    ElMessage.error("如果收藏命令,必须输入收藏名称");
    return;
  }

  axios.post<ResponseData>(`/api/cmd_note/`, { "cmd_name": cmd.name, "cmd_data": cmd.data })
    .then((ret) => {
      if (ret.data.code === 0) {
        ElMessage.success("收藏成功");
        getAllCmdNote();
      } else {
        ElMessage.error("收藏命令出错了");
      }
    });

}

/**
 * 删除命令收藏
 */
function delCmdNote(id: number) {
  axios.delete<ResponseData>(`/api/cmd_note/${id}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        cmdNotes.value = ret.data.data;
        ElMessage.success("删除成功");
      } else {
        ElMessage.error("删除命令收藏出错了");
      }
    });
}

/**
 * 更新命令收藏
 */
function putCmdNote(id: number) {

}

/**
 * 查询所有命令收藏
 */
function getAllCmdNote() {
  axios.get<ResponseData>("/api/cmd_note").then((ret) => {
    if (ret.data.code === 0) {
      cmdNotes.value = ret.data.data;
    } else {
      ElMessage.error("获取主机列表错误");
    }
  });
}

/**
 * 在当前会话执行收藏命令
 */
function execCmdCurrentSession(row: CmdNode) {
  try {
    data.current_host.ws.send(row.cmd_data + "\n");
  } catch (e) {
    ElMessage.error("当前会话执行命令失败");
  }
}

/**
 * 在所有会话执行收藏命令
 */
function execCmdAllSession(row: CmdNode) {
  try {
    if (data.host_tabs.length === 0) {
      ElMessage.error("没有连接会话");
      return;
    }
    data.host_tabs.forEach((h) => {
      h.ws.send(row.cmd_data + "\n");
    });
  } catch (e) {
    ElMessage.error("执行命令失败");
  }
}

/**
 * 添加密钥文件
 */
function addCertFile() {
  const input = document.createElement("input");
  input.type = "file";
  input.addEventListener("change", (event) => {
    const files = (event.target as HTMLInputElement).files;
    if (files && files.length > 0) {
      let certFile = files[0];
      const isLt1M = certFile.size / 1024 / 1024 < 1;
      if (!isLt1M) {
        ElMessage.error("上传文件大小不能超过 1MB!");
        return;
      }
      const reader = new FileReader();
      reader.onload = (e) => {
        data.cert_data = (e.target as FileReader).result as string;
      };
      reader.readAsText(certFile);
    }
  });
  input.click();
}

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

  if (data.auth_type === "pwd" && data.pwd.length === 0) {
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

  if (data.auth_type === "cert" && data.cert_data === "") {
    ElMessage.error("使用密钥登陆,密钥内容不能为空");
    return verifyFromData;
  }

  let h = {
    id: data.id,
    name: data.name,
    address: data.address,
    user: data.user,
    auth_type: data.auth_type,
    net_type: data.net_type,
    cert_data: data.cert_data,
    cert_pwd: data.cert_pwd,
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
    pty_type: data.pty_type,
    init_cmd: data.init_cmd,
    init_banner: data.init_banner,
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
  data.auth_type = "pwd";
  data.net_type = "tcp4";
  data.cert_data = "";
  data.cert_pwd = "";
  data.port = 22;
  data.session_id = "";
  data.background = "#000000";
  data.foreground = "#FFFFFF";
  data.cursor_color = "#FFFFFF";
  data.font_family = "Courier";
  data.font_size = 16;
  data.cursor_style = "block";
  data.shell = "bash";
  data.pty_type = "xterm-256color";
  data.init_cmd = "";
  data.init_banner = "";
  data.host_config_collapse = ['1'];
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

  let formData = new FormData();
  formData.append("session_id", host.session_id);
  formData.append("path", dir);
  axios.post<ResponseData>("/api/sftp/list", formData).then((ret) => {
    if (ret.data.code === 0) {
      data.dir_info = ret.data.data;
      data.sftp_current_dir = dir;
    } else {
      ElMessage.error("获取文件列表错误");
    }
  });
}

/**
 * 上传文件
 */
function uploadFile(path: string) {
  data.sftp_upload_percentage = 0;
  function upload(fileList: FileList) {
    let formData = new FormData();
    formData.append("session_id", data.current_host.session_id);
    formData.append("path", path);
    for (let i = 0; i < fileList.length; i++) {
      formData.append("files", fileList[i]);
    }

    axios({
      url: '/api/sftp/upload',
      method: 'put',
      data: formData,
      //上传进度
      onUploadProgress: (progressEvent: AxiosProgressEvent) => {
        const { loaded, total } = progressEvent;
        if (!total) {
          // 没有获取到总大小，可能是流式上传或者chunked传输
          data.sftp_upload_percentage = loaded;
        } else {
          // 计算进度，可以用 loaded / total 得到一个0到1的数字
          data.sftp_upload_percentage = loaded / total * 100 | 0;
        }
      }
    }).then((ret) => {
      if (ret.data.code === 0) {
        // ElMessage.success(ret.data.msg);
        listDir(data.sftp_current_dir, data.current_host);
        let list = ret.data.data as Array<string>;
        if (list) {
          let msg = "";
          list.forEach((i) => {
            msg += `<p>${i}</p>`;
          });
          ElNotification({
            type: 'success',
            duration: 7000,
            title: ret.data.msg,
            dangerouslyUseHTMLString: true,
            message: msg,
          });
        }
      } else {
        ElMessage.error("上传失败");
      }
    }).catch(() => {
      ElMessage.error("上传异常");
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
function downloadFile(file: FileInfo) {
  /*
  // POST 方式
  let formData = new FormData();
  formData.append("session_id", data.current_host.session_id);
  formData.append("path", file.path);
  axios.post<Blob>("/api/sftp/download", formData).then((ret) => {
    let blob = new Blob([ret.data], { type: 'application/x-download' });
    let a = document.createElement("a");
    a.style.display = 'none';
    let url = window.URL.createObjectURL(blob);
    a.href = url;
    a.download = file.name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a); 
    window.URL.revokeObjectURL(url);
  });
  */
  let reqUrl = `/api/sftp/download?Authorization=${localStorage.getItem("token")}&session_id=${data.current_host.session_id}&path=${encodeURIComponent(file.path).replace(/%/g, "%25")}`;
  let a = document.createElement("a");
  a.style.display = 'none';
  a.href = reqUrl;
  a.download = file.name;
  a.click();
}

/**
 * SFTP文件删除
 */
function deleteFile(file: FileInfo) {
  let body = {
    "session_id": data.current_host.session_id,
    "path": file.path
  }
  axios.delete<ResponseData>("/api/sftp/delete", { data: body }).then((ret) => {
    if (ret.data.code === 0) {
      listDir(data.sftp_current_dir, data.current_host);
      ElMessage.success("删除文件成功");
    } else {
      ElMessage.error("删除文件出错了");
    }
  });
}

/**
 * SFTP创建目录
 */
function createDir(dir: string, h: Host) {
  let body = {
    "session_id": h.session_id,
    "path": dir
  }
  axios.post<ResponseData>("/api/sftp/create_dir", body).then((ret) => {
    if (ret.data.code === 0) {
      listDir(data.sftp_current_dir, data.current_host);
      ElMessage.success("创建目录成功");
    } else {
      ElMessage.error("创建目录出错了");
    }
  });
}

/**
 * 获取所有主机列表
 */
function getAllHost() {
  axios.get<ResponseData>("/api/conn_conf").then((ret) => {
    if (ret.data.code === 0) {
      data.host_list = ret.data.data;
    } else {
      ElMessage.error("获取主机列表错误");
    }
  })
}

/**
 * 创建或更新主机
 */
function createOrUpdateHost(host: Host, m: Mode) {
  // 关闭模态框,from 表单验证后续在搞 :rules="host_from_rules"
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
  if (m == 0) {
    // 新增
    axios.post<ResponseData>("/api/conn_conf", host)
      .then((ret) => {
        if (ret.data.code === 0) {
          data.host_list = ret.data.data;
          cleanFrom();
        } else {
          ElMessage.error("新增出错了");
        }
      })
  } else {
    // 更新
    axios.put<ResponseData>(`/api/conn_conf`, host)
      .then((ret) => {
        if (ret.data.code === 0) {
          data.host_list = ret.data.data;
          cleanFrom();
        }
        else {
          ElMessage.error("更新出错了");
        }
      })
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
  data.auth_type = row.auth_type;
  data.net_type = row.net_type;
  data.cert_data = row.cert_data;
  data.cert_pwd = row.cert_pwd;
  data.pwd = row.pwd;
  data.port = row.port;
  data.background = row.background;
  data.foreground = row.foreground;
  data.cursor_color = row.cursor_color;
  data.font_family = row.font_family;
  data.font_size = row.font_size;
  data.cursor_style = row.cursor_style;
  data.shell = row.shell;
  data.pty_type = row.pty_type;
  data.init_cmd = row.init_cmd;
  data.init_banner = row.init_banner;
  data.host_config_collapse = ['1'];
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
  axios.delete<ResponseData>(`/api/conn_conf/${row.id}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.host_list = ret.data.data;
        cleanFrom();
      } else {
        ElMessage.error("删除主机出错了");
      }
    });
}

/**
 * 去掉几个引用对象属性
 * @param data 
 */
function getHost(data: Host): Omit<Host, 'fit' | 'term' | 'ws' | 'is_close'> {
  if (data.term) {
    try {
      data.fit.dispose();
      data.term.dispose();
      data.ws.close();
    } catch (err) {
      console.log("清理资源错误:" + err);
    }
  }

  let connectTabElement = document.getElementById(data.session_id);
  if (connectTabElement) {
    connectTabElement.innerHTML = "";
  }

  return {
    id: data.id,
    name: data.name,
    address: data.address,
    user: data.user,
    auth_type: data.auth_type,
    net_type: data.net_type,
    cert_data: data.cert_data,
    cert_pwd: data.cert_pwd,
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
    pty_type: data.pty_type,
    init_cmd: data.init_cmd,
    init_banner: data.init_banner,
  };
}

/**
 * 连接已经保存过的主机
 */
function connectHost(host: Host, isReconnect: boolean = false) {
  // 关闭模态框
  data.host_dialog_visible = false;

  let requestUrl = "/api/ssh/create_session";
  // 如果重连,在url加上会话ID
  if (isReconnect) {
    requestUrl += `?session_id=${host.session_id}`;
  }

  // 上一个版本的解包
  let connHost = getHost(host) as Host;
  axios.post<ResponseData>(requestUrl, connHost)
    .then((ret) => {
      if (ret.data.code === 0) {
        let session_id = ret.data.data;
        connHost.session_id = session_id;

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

        // 如果是重连就不需要再建立tab页面,直接替换
        if (isReconnect) {
          for (let [index, h] of data.host_tabs.entries()) {
            if (h.session_id === session_id) {
              connHost.is_close = false;
              data.host_tabs[index] = connHost;
              break;
            }
          }
        } else {
          // 新连接添加tab 页面
          data.host_tabs.push(connHost);
        }

        nextTick(() => {
          let connectTabElement = document.getElementById(connHost.session_id);

          if (connectTabElement === null) {
            ElMessage.error("创建连接获取dom为空!");
            return;
          }

          connectTabElement.style.height = Math.floor(window.innerHeight - 70) + "px";
          connHost.term.open(connectTabElement);
          connHost.fit.fit();

          let param = `h=${connHost.term.rows}&w=${connHost.term.cols}&session_id=${connHost.session_id}&Authorization=${localStorage.getItem("token")}`;
          let headPart = `${location.protocol == "http:" ? "ws://" : "wss://"}${location.host}`;
          let tailPart = `/api/ssh/conn?${param}`;

          let basePath = window.location.pathname.replace("/app/", "");
          if (import.meta.env.VITE_ROUTE_MODE === "WebHistory") {
              if (import.meta.env.VITE_WEB_BASE_DIR) {
                  basePath = `${import.meta.env.VITE_WEB_BASE_DIR}`;
              } else {
                  basePath = "";
              }
          }

          let webSockerUrl = `${headPart}${basePath}${tailPart}`;

          let ws = new WebSocket(webSockerUrl);
          ws.onopen = function () {
            try {
              // 初始化benner
              let bannerStr = connHost.init_banner.trim();
              if (bannerStr !== "") {
                connHost.term.writeln(bannerStr);
              }

              // 调整窗口大小
              windowResize();

              // 初始化命令
              let cmdStr = connHost.init_cmd.trim()
              if (cmdStr !== "") {
                ws.send(`${cmdStr}\n`)
              }
            } catch (err) {
              console.log(err);
            }
          }

          ws.onerror = function (err) {
            console.log("WebSocket error");
            connHost.term.writeln("##  连接出错,请重连!  ##");
          }

          ws.onclose = function () {
            console.log("WebSocket close:" + connHost.session_id);
            connHost.term.writeln("##  连接关闭,请重连!  ##");
            connHost.is_close = true;
            if (data.current_host.session_id === session_id) {
              data.current_host.is_close = true;
            }
          }

          connHost.term.loadAddon(new AttachAddon(ws));
          connHost.ws = ws;
          connHost.is_close = false;
          connHost.term.focus();
          // 清空 from 表单数据
          cleanFrom();

          // 设置当前激活的host
          data.current_host = { ...connHost };
        });
      } else {
        ElMessage.error("创建连接出错了");
      }
    }).catch((err) => {
      ElMessage.error("创建会话出错了");
      console.log(err)
    });
}

/**
 * 删除tab
 */
function removeTab(tabId: string | number) {
  try {
    axios.post(`/api/ssh/disconnect?session_id=${tabId}`);
  } catch (error) {
    console.log(error);
  }

  let removeIndex = 0;
  for (let [index, h] of data.host_tabs.entries()) {
    if (h.session_id === String(tabId)) {
      removeIndex = index;
      break;
    }
  }

  // 销毁term 对象
  data.host_tabs[removeIndex].fit.dispose();
  data.host_tabs[removeIndex].term.dispose();
  data.host_tabs[removeIndex].ws.close();

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
    let activeHost = { ...data.host_tabs[removeIndex - 1] };
    setCurrentAcitveHost(activeHost.session_id);
  }
}

/***
 * 点击切换tab
 */
function selectTab(tab: any) {
  let sessionId = tab.props.name;
  if (data.current_host.session_id === sessionId) {
    // 激活的已经是当前窗口直接返回
    return;
  }
  setCurrentAcitveHost(sessionId);
}

/**
 * 设置当前正在使用的主机
 */
function setCurrentAcitveHost(sessionId: string) {
  for (const host of data.host_tabs) {
    if (host.session_id === sessionId) {
      data.current_host = { ...host };
      break;
    }
  }
  windowResize();
}

/**
 * 更改窗口大小
 */
function windowResize() {
  let currentHost = data.current_host;
  if (currentHost.session_id === "") {
    return;
  }
  // 没有在主机连接路由页面
  if (router.currentRoute.value.name !== "Home") {
    return;
  }
  nextTick(() => {
    let connectTabElement = document.getElementById(currentHost.session_id);
    if (connectTabElement === null) {
      console.log("调整窗口大小,没有获取到dom");
      return;
    }
    (connectTabElement as HTMLElement).style.height = Math.floor(window.innerHeight - 70) + "px";

    currentHost.fit.fit();
    //if (data.h !== currentHost.term.rows || data.w !== currentHost.term.cols) {
    let url = `/api/ssh/conn?w=${currentHost.term.cols}&h=${currentHost.term.rows}&session_id=${currentHost.session_id}`;
    axios.patch<ResponseData>(url)
    //}

    data.h = Math.floor(currentHost.term.rows);
    data.w = Math.floor(currentHost.term.cols);
  });
}

/**
 * 报告连接状态
 */
function reportConnectStatus() {
  statusSetInterval = setInterval(() => {
    let fm = new FormData();
    data.host_tabs.forEach((hont) => {
      fm.append("ids", hont.session_id);
    });
    axios.put<ResponseData>("/api/conn_manage/refresh_conn_time", fm)
      .then((res) => {
        if (res.data.code !== 0) {
          console.log("刷新失败");
        }
      });
  }, 10000);
}

/**
 * 跳转到管理页面
 */
function toManage() {
  //router.push({ name: "Manage" });
  data.manage_dialog_visible = true;
}

/**
 * 防抖
 * @param fn 
 * @param delay 
 */
function debounce(fn: Function, delay: number) {
  let timer = 0;
  return function (event: Event) {
    clearTimeout(timer);
    timer = setTimeout(() => {
      fn();
    }, delay)
  }
}

/**
 * 节流
 * @param fn 
 * @param delay 
 */
function throttle(fn: Function, delay: number) {
  let record = Date.now();
  return function (event: Event) {
    let now = Date.now();
    if (now - record > delay) {
      fn();
      record = now;
    }
  }
}

/**
 * 断开所有会话
 */
function disconnectAllSession() {
  // 清理连接资源
  data.host_tabs.forEach((host, index) => {
    try {
      axios.post(`/api/ssh/disconnect?session_id=${host.session_id}`);
    } catch (error) {
      console.log(error);
    }
  });
}

/**
 * 退出登陆
 */
function logout() {
  disconnectAllSession();
  globalStore.logout();
  localStorage.setItem("auth", "no");
  router.push({ "name": "Login" });
}

/**
 * 挂载后执行
 */
onMounted(() => {
  router = useRouter();
  reportConnectStatus();
  getAllHost();
  getAllCmdNote();
  window.addEventListener("resize", debounce(windowResize, 200));
  windowResize();
  window.onbeforeunload = function () {
    return "关闭吗";
  };
});

/**
 * 销毁前执行
 */
onBeforeUnmount(() => {
  clearInterval(statusSetInterval);
  disconnectAllSession();
  window.onbeforeunload = null;
})

// 添加计算属性获取终端背景色
const terminalBackground = computed(() => {
  if (data.current_host?.term?.options?.theme?.background) {
    return data.current_host.term.options.theme.background;
  }
  return data.background || '#000000'; // 使用配置的背景色或默认黑色
});

</script>


<style scoped>
.file-dialog {
  margin-top: 0px;
}

:deep(.el-tabs__header.is-top) {
  margin: 0;
  border-bottom: none;
}

/* 优化 tabs 整体布局 */
:deep(.el-tabs) {
  height: calc(100vh - 32px);
}

:deep(.el-tabs__content) {
  flex: 1;
  margin-top: 1px;
  overflow: hidden;
}

:deep(.el-tab-pane) {
  height: 100%;
}

/* 确保终端容器填充剩余空间 */
:deep(.el-tab-pane > div) {
  height: 100%;
}

/* 动态背景色样式 */
:deep(.el-tabs__content),
:deep(.el-tab-pane),
:deep(.el-tab-pane > div > #term-data) {
  background-color: v-bind('terminalBackground');
}
</style>
