<template>
  <router-view/>
</template>

<style>

.el-header, .el-footer {
  background-color: #2b75d6;
} 

/* tab 标签和终端之间的空白处理 */
#app > section > div > div > div.el-tabs__header.is-top {
  margin-bottom: 0px;
}

</style>
