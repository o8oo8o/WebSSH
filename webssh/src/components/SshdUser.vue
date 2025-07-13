<template>
  <el-tab-pane label="SSHD账号管理" name="sshdUser">
    <!-- ========================================= -->
    <el-card>
      <el-row>
        <el-table :data="data.user_list" style="width: 100%" :show-overflow-tooltip="true">
          <el-table-column fixed sortable prop="name" label="用户名"></el-table-column>
          <el-table-column sortable prop="desc_info" label="备注信息" width="200"></el-table-column>
          <el-table-column sortable prop="is_enable" label="是否启用"></el-table-column>
          <el-table-column sortable prop="expiry_at" label="过期时间" width="180"></el-table-column>
          <el-table-column fixed="right" label="操作">
            <template #header>
              <el-button type="primary" @click="addUser">新增</el-button>
              <el-button type="primary" @click="getUserList(0, 10000)">刷新</el-button>
            </template>
            <template #default="scope">
              <el-button type="success" @click="editUser(scope.row)">编辑</el-button>
              <el-popconfirm confirmButtonText="删除" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                title="确定删除吗" @confirm="deleteUserById(scope.row.id)">
                <template #reference>
                  <el-button type="danger">删除</el-button>
                </template>
              </el-popconfirm>

            </template>
          </el-table-column>
        </el-table>
      </el-row>
    </el-card>
    <!-- ========================================= -->
    <el-dialog title="SSHD账号信息" v-model="data.user_dialog_visible" width="60%">
      <el-form label-width="80px">
        <el-form-item label="备注名" prop="desc_info">
          <el-input minlength="1" maxlength="60" v-model.trim="user.desc_info" show-word-limit
            placeholder="请输入备注"></el-input>
        </el-form-item>
        <el-form-item label="用户名" prop="name">
          <el-input v-model.trim="user.name" minlength="1" maxlength="30" show-word-limit
            placeholder="请输入用户名"></el-input>
        </el-form-item>
        <el-form-item label="密码" prop="pwd">
          <el-input minlength="1" maxlength="60" v-model.trim="user.pwd" type="passrowd" show-password show-word-limit
            placeholder="用户密码"></el-input>
        </el-form-item>
        <el-form-item label="启用" prop="is_enable">
          <el-radio-group v-model="user.is_enable" id="is_enable">
            <el-radio value="Y">是</el-radio>
            <el-radio value="N">否</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="user.expiry_at" type="datetime" placeholder="选择账号过期时间" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="data.user_dialog_visible = false">取消</el-button>
          <el-button type="success" @click="saveUser">保存</el-button>
        </span>
      </template>
    </el-dialog>
    <!-- ========================================= -->
  </el-tab-pane>
</template>

<script setup lang="ts">
import { onMounted, reactive } from "vue";
import { ElMessage, dayjs } from "element-plus";
import axios from "axios";


enum Mode {
  "create" = 0,
  "update" = 1,
}

/**
 * 用户信息
 */

interface User {
  id: number;
  name: string;
  pwd: string;
  desc_info: string;
  is_enable: "Y" | "N";
  expiry_at: Date;
}

interface UserInfo extends User {
  created_at?: string;
  updated_at?: string;
}

interface ResponseData {
  code: number;
  msg: string;
  data?: any
}

let data = reactive({
  mode: Mode.create,
  user_dialog_visible: false,
  active_name: "sshdUser",
  user_list: Array<UserInfo>(),
  is_admin: true,
});

let user = reactive<User>({
  id: 0,
  name: "",
  pwd: "",
  desc_info: "",
  is_enable: "Y",
  expiry_at: new Date(2050, 0, 0, 0, 0, 0)
});


/**
 * 获取sshd用户列表
 */
function getUserList(offset: number = 0, limit: number = 10000) {
  axios.get<ResponseData>(`/api/sshd_user?offset=${offset}&limit=${limit}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.user_list = ret.data.data;
      } else {
        ElMessage.error("获取sshd用户列表错误");
      }
    })
}

/**
 * 添加用户
 */
function addUser() {
  user.id = 0;
  user.name = "";
  user.pwd = "";
  user.desc_info = "";
  user.is_enable = "Y";
  user.expiry_at = new Date(2050, 0, 0, 0, 0, 0);
  data.user_dialog_visible = true;

}

/**
 * 编辑用户
 * @param u 
 */
function editUser(u: UserInfo) {
  user.id = u.id;
  user.name = u.name;
  user.pwd = u.pwd;
  user.desc_info = u.desc_info;
  user.is_enable = u.is_enable;
  user.expiry_at = u.expiry_at;
  data.user_dialog_visible = true;
}


function getRequestBody(): any {
  return {
    id: user.id,
    name: user.name,
    pwd: user.pwd,
    desc_info: user.desc_info,
    is_enable: user.is_enable,
    expiry_at: dayjs(user.expiry_at).format("YYYY-MM-DD HH:mm:ss"),
  }
}

/**
 * 根据ID更新用户
 * @param userId ID
 */
function updateUserById() {
  axios.put<ResponseData>(`/api/sshd_user`, getRequestBody())
    .then((ret) => {
      if (ret.data.code === 0) {
        data.user_list = ret.data.data;
      } else {
        ElMessage.error("更新sshd用户错误,请检查输入");
      }
    })
}

/**
 * 根据ID删除用户
 * @param userId ID
 */
function deleteUserById(userId: number) {
  axios.delete<ResponseData>(`/api/sshd_user/${userId}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.user_list = ret.data.data;
      } else {
        ElMessage.error("删除sshd用户错误");
      }
    })
}

/**
 * 创建用户
 */
function createUser() {
  axios.post<ResponseData>(`/api/sshd_user`, getRequestBody())
    .then((ret) => {
      if (ret.data.code === 0) {
        data.user_list = ret.data.data;
      } else {
        ElMessage.error("创建sshd用户错误,请检查输入");
      }
    })
}

function saveUser() {
  if (user.id === 0) {
    createUser();
  } else {
    updateUserById()
  }
  user.id = 0;
  user.name = "";
  user.pwd = "";
  user.desc_info = "";
  user.is_enable = "Y";
  user.expiry_at = new Date(2050, 0, 0, 0, 0, 0);
  data.user_dialog_visible = false;
}


onMounted(() => {
  getUserList();
})


</script>


<style scoped></style>