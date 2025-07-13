<template>
  <div class="container">
    <div class="github-links">
      <el-link type="primary" href="https://github.com/o8oo8o/WebSSH" target="_blank">
        <div class="github-link">
          <i class="github-icon">
            <svg viewBox="0 0 16 16" width="64" height="64">
              <path fill="currentColor"
                d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
            </svg>
          </i>
          <span class="github-text">GitHub</span>
        </div>
      </el-link>
    </div>
    <div class="text3d-container">
      <div class="text3d">
        <span>W</span>
        <span>e</span>
        <span>b</span>
        <span>&nbsp;</span>
        <span>S</span>
        <span>S</span>
        <span>H</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.container {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  min-height: calc(100vh - 100px);
  width: 100%;
  perspective: 1000px;
  position: relative;
  gap: 100px;
}

.github-links {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
  z-index: 2;
  position: relative;
  margin-top: 50px;
}

.github-link {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px;
  border-radius: 8px;
  transition: all 0.3s ease;
  font-size: 18px;
  background-color: rgba(64, 158, 255, 0.05);
}

.github-link:hover {
  background-color: rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}

.github-icon {
  display: flex;
  align-items: center;
  color: #409EFF;
}

.github-text {
  font-weight: 500;
  letter-spacing: 0.5px;
}

/* 3D 文字效果 */
.text3d-container {
  position: relative;
  z-index: 1;
  margin-top: 20px;
  text-align: center;
}

.text3d {
  font-size: 5em;
  font-weight: bold;
  font-family: Arial, sans-serif;
  color: #409EFF;
  display: flex;
  justify-content: center;
}

.text3d span {
  display: inline-block;
  text-shadow: 
    0 1px 0 #ccc,
    0 2px 0 #c9c9c9,
    0 3px 0 #bbb,
    0 4px 0 #b9b9b9,
    0 5px 0 #aaa,
    0 6px 1px rgba(0,0,0,.1),
    0 0 5px rgba(0,0,0,.1),
    0 1px 3px rgba(0,0,0,.3),
    0 3px 5px rgba(0,0,0,.2),
    0 5px 10px rgba(0,0,0,.25),
    0 10px 10px rgba(0,0,0,.2),
    0 20px 20px rgba(0,0,0,.15);
  animation: popUp 2s ease-in-out infinite;
}

/* 为每个字母设置不同的动画延迟 */
.text3d span:nth-child(1) { animation-delay: 0s; }
.text3d span:nth-child(2) { animation-delay: 0.2s; }
.text3d span:nth-child(3) { animation-delay: 0.4s; }
.text3d span:nth-child(4) { animation-delay: 0.6s; }
.text3d span:nth-child(5) { animation-delay: 0.8s; }
.text3d span:nth-child(6) { animation-delay: 1s; }

@keyframes popUp {
  0% {
    transform: translateY(0) scale(1);
  }
  25% {
    transform: translateY(-20px) scale(1.1);
  }
  50% {
    transform: translateY(0) scale(1);
  }
  75% {
    transform: translateY(10px) scale(0.9);
  }
  100% {
    transform: translateY(0) scale(1);
  }
}

/* 修改悬浮效果 */
.text3d span:hover {
  animation-play-state: paused;
  color: #66b1ff;
}

/* 响应式设计 */
@media screen and (max-width: 768px) {
  .container {
    gap: 50px;
  }

  .github-links {
    margin-top: 30px;
  }

  .text3d {
    font-size: 3em;
  }
}

@media screen and (max-width: 480px) {
  .container {
    gap: 30px;
  }

  .github-links {
    margin-top: 20px;
  }

  .text3d {
    font-size: 2em;
  }

  .text3d span {
    text-shadow: 
      0 1px 0 #ccc,
      0 2px 0 #c9c9c9,
      0 3px 0 #bbb,
      0 4px 5px rgba(0,0,0,.15);
  }
}

/* 暗色模式支持 */
@media (prefers-color-scheme: dark) {
  .github-link {
    background-color: rgba(64, 158, 255, 0.1);
  }
  
  .github-link:hover {
    background-color: rgba(64, 158, 255, 0.2);
  }
  
  .text3d {
    color: #60aeff;
  }
}
</style>