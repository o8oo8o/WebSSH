<template></template>
<script setup lang="ts">
import { useRouter } from "vue-router";
let router = useRouter();
console.log("404 Page");
router.push({ "name": "Home" })
</script>
