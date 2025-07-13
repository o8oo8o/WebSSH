<template>
  <el-tab-pane label="SSHD证书管理" name="sshdCert">
    <!-- ========================================= -->
    <el-card>
      <el-row>
        <el-table :data="data.cert_list" style="width: 100%" :show-overflow-tooltip="true">
          <el-table-column fixed sortable prop="name" label="标题"></el-table-column>
          <el-table-column sortable prop="desc_info" label="备注" width="200"></el-table-column>
          <el-table-column sortable prop="is_enable" label="是否启用"></el-table-column>
          <el-table-column sortable prop="expiry_at" label="过期时间" width="180"></el-table-column>
          <el-table-column fixed="right" label="操作">
            <template #header>
              <el-button type="primary" @click="addCert">新增</el-button>
              <el-button type="primary" @click="getCertList(0, 10000)">刷新</el-button>
            </template>
            <template #default="scope">
              <el-button type="success" @click="editCert(scope.row)">编辑</el-button>
              <el-popconfirm confirmButtonText="删除" cancelButtonText="取消" icon="el-icon-info" iconColor="red"
                title="确定删除吗" @confirm="deleteCertById(scope.row.id)">
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
    <el-dialog title="SSHD证书信息" v-model="data.cert_dialog_visible" width="80%">
      <el-form label-width="80px">
        <el-form-item label="备注" prop="desc_info">
          <el-input minlength="1" maxlength="60" v-model.trim="cert.desc_info" show-word-limit
            placeholder="请输入备注"></el-input>
        </el-form-item>
        <el-form-item label="标题" prop="name">
          <el-input v-model.trim="cert.name" minlength="1" maxlength="30" show-word-limit
            placeholder="请输入标题"></el-input>
        </el-form-item>
        <el-form-item label="公钥" prop="pub_key">
          <el-input v-model.trim="cert.pub_key" type="textarea" :row="3" minlength="16"
            placeholder="支持以 'ssh-rsa', 'ssh-ed25519' 开头" />
        </el-form-item>
        <el-form-item label="启用" prop="is_enable">
          <el-radio-group v-model="cert.is_enable" id="is_enable">
            <el-radio value="Y">是</el-radio>
            <el-radio value="N">否</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="cert.expiry_at" type="datetime" placeholder="选择证书过期时间" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="data.cert_dialog_visible = false">取消</el-button>
          <el-button type="success" @click="saveCert">保存</el-button>
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
 * 证书信息
 */

interface Cert {
  id: number;
  name: string;
  pub_key: string;
  desc_info: string;
  is_enable: "Y" | "N";
  expiry_at: Date;
}

interface CertInfo extends Cert {
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
  cert_dialog_visible: false,
  active_name: "sshdCert",
  cert_list: Array<CertInfo>(),
  is_admin: true,
});

let cert = reactive<Cert>({
  id: 0,
  name: "",
  pub_key: "",
  desc_info: "",
  is_enable: "Y",
  expiry_at: new Date(2050, 0, 0, 0, 0, 0)
});


/**
 * 获取sshd证书列表
 */
function getCertList(offset: number = 0, limit: number = 10000) {
  axios.get<ResponseData>(`/api/sshd_cert?offset=${offset}&limit=${limit}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.cert_list = ret.data.data;
      } else {
        ElMessage.error("获取sshd证书列表错误");
      }
    })
}

/**
 * 添加sshd证书
 */
function addCert() {
  cert.id = 0;
  cert.name = "";
  cert.pub_key = "";
  cert.desc_info = "";
  cert.is_enable = "Y";
  cert.expiry_at = new Date(2050, 0, 0, 0, 0, 0);
  data.cert_dialog_visible = true;

}

/**
 * 编辑sshd证书
 * @param u 
 */
function editCert(u: CertInfo) {
  cert.id = u.id;
  cert.name = u.name;
  cert.pub_key = u.pub_key;
  cert.desc_info = u.desc_info;
  cert.is_enable = u.is_enable;
  cert.expiry_at = u.expiry_at;
  data.cert_dialog_visible = true;
}


function getRequestBody(): any {
  return {
    id: cert.id,
    name: cert.name,
    pub_key: cert.pub_key,
    desc_info: cert.desc_info,
    is_enable: cert.is_enable,
    expiry_at: dayjs(cert.expiry_at).format("YYYY-MM-DD HH:mm:ss"),
  }
}

/**
 * 根据ID更新sshd证书
 * @param certId ID
 */
function updateCertById() {
  axios.put<ResponseData>(`/api/sshd_cert`, getRequestBody())
    .then((ret) => {
      if (ret.data.code === 0) {
        data.cert_list = ret.data.data;
      } else {
        ElMessage.error("更新sshd证书错误,请检查输入");
      }
    })
}

/**
 * 根据ID删除sshd证书
 * @param certId ID
 */
function deleteCertById(certId: number) {
  axios.delete<ResponseData>(`/api/sshd_cert/${certId}`)
    .then((ret) => {
      if (ret.data.code === 0) {
        data.cert_list = ret.data.data;
      } else {
        ElMessage.error("删除sshd证书错误");
      }
    })
}

/**
 * 创建sshd证书
 */
function createCert() {
  axios.post<ResponseData>(`/api/sshd_cert`, getRequestBody())
    .then((ret) => {
      if (ret.data.code === 0) {
        data.cert_list = ret.data.data;
      } else {
        ElMessage.error("创建sshd证书错误,请检查输入");
      }
    })
}

function saveCert() {
  if (cert.id === 0) {
    createCert();
  } else {
    updateCertById()
  }
  cert.id = 0;
  cert.name = "";
  cert.pub_key = "";
  cert.desc_info = "";
  cert.is_enable = "Y";
  cert.expiry_at = new Date(2050, 0, 0, 0, 0, 0);
  data.cert_dialog_visible = false;
}


onMounted(() => {
  getCertList();
})

</script>


<style scoped></style>