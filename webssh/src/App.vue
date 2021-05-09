<template>
  <!-- <div id="nav">
    <router-link to="/">Home</router-link> |
    <router-link to="/about">About</router-link>
  </div> -->
  <router-view/>
</template>

<style>

.el-header, .el-footer {
  background-color: #2b75d6;
  /* color: rgb(156, 73, 73); */
  /* line-height: 40px; */
} 


/* tab 标签和终端之间的空白处理 */
#app > section > div > div > div.el-tabs__header.is-top {
  margin-bottom: 0px;
  /* height: 28px; */
}

/* #app > section > div > div > div.el-tabs__header.is-top > div  {
  height: 28px;
} */

</style>
