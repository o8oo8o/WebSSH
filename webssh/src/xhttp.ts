/**
 * 所有的Hook函数,必须又一个boolean 的返回值,当返回值为false的时候,执行下一个钩子函数,当返回值为true的时候,终止所有,直接返回
Example

// -----------------------------------------
// filename: main.ts

import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';

import xhttp from "./xhttp";

xhttp.addGlobalRequestHook((request) => {
  request.headers.append("token", "token_message");
  return {"request":request,"isBreak":false};
});

createApp(App).use(store).use(router).mount('#app');


// -----------------------------------------
// filename: About.vue

<template>
  <div class="about">
    <h1 @click="fa">Fa</h1>
    <h1 @click="fb">Fb</h1>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import xhttp, { Xhttp } from "../xhttp";

function fa() {
  xhttp.get("/api/home?fa").then((ret) => {
    console.log(ret);
  });
}

function fb() {
  let xhttp = new Xhttp();
  xhttp.get("/api/home?fb").then((ret) => {
    console.log(ret);
  });
}

export default defineComponent({
  name: "about",
  setup() {
    return {
      fa,
      fb,
    };
  },
});
</script>

// -----------------------------------------
 */

interface header {
    name: string;
    value: string
}

interface RequestHookResult {
    request: Request;
    isBreak: boolean
}

interface ResponseHookResult {
    response: Response;
    isBreak: boolean
}

interface requestHook {
    (request: Request): RequestHookResult;
}

interface responseHook {
    (response: Response): ResponseHookResult
}


class Xhttp {
    private static instance: Xhttp;

    private headers = new Headers();

    private requestInit: RequestInit = {
        method: "GET",
        headers: this.getHeaders(),
        mode: "cors",
        cache: "default",
        credentials: "include",
    };

    private static globalRequestHooks: Array<requestHook> = [];
    private static globalResponseHooks: Array<responseHook> = [];
    private requestHooks: Array<requestHook> = [];
    private responseHooks: Array<responseHook> = [];

    public constructor(init?: RequestInit) {
        // console.log(`xhttp_constructor`);
        if (init) {
            this.requestInit = init;
        }
    }
    /**
     * 通过init静态方法创建Xhttp实例
     * @param init
     */
    public static new(init?: RequestInit): Xhttp {
        return new Xhttp(init);
    };

    /**
     * 调用request回调函数
     * @param request 
     */
    private callRequestHook(request: Request): RequestHookResult {
        let tmpRequest = request;
        let result: RequestHookResult;
        for (let i = 0; i < Xhttp.globalRequestHooks.length; i++) {
            result = Xhttp.globalRequestHooks[i](tmpRequest);
            tmpRequest = result.request;

            if (result.isBreak) {
                return result;
            }
        }


        for (let i = 0; i < this.requestHooks.length; i++) {
            result = this.requestHooks[i](tmpRequest);
            tmpRequest = result.request;
            if (result.isBreak) {
                return result;
            }
        }
        return { "request": tmpRequest, "isBreak": false };
    }

    /**
     * 调用response回调函数
     * @param response 
     */
    private callResponseHook(response: Response): ResponseHookResult {
        let tmpResponse = response;
        let result: ResponseHookResult;
        for (let i = 0; i < Xhttp.globalResponseHooks.length; i++) {
            result = Xhttp.globalResponseHooks[i](tmpResponse);
            tmpResponse = result.response;
            if (result.isBreak) {
                return result;
            }
        }

        for (let i = 0; i < this.responseHooks.length; i++) {
            result = this.responseHooks[i](response);
            tmpResponse = result.response;
            if (result.isBreak) {
                return result;
            }
        }
        return { "response": tmpResponse, "isBreak": false };;
    }


    /**
     * 单例模式
     */
    // public static getInstance() {
    //     if (!Xhttp.instance) {
    //         Xhttp.instance = new Xhttp();
    //     }
    //     return Xhttp.instance;
    // }

    private static getInstance(): Xhttp {
        return new Xhttp();
    }

    /**
     * 获取请求Headers
     */
    private getHeaders(): Headers {
        return this.headers;
    }

    /**
     * 同时设置多个header
     * @param headers 
     */

    public setHeaders(headers: Array<header>): void {
        headers.forEach((header) => {
            this.headers.set(header.name, header.value);
        });
    }

    /**
     * 设置单个header
     * @param name 
     * @param value 
     */
    public setHeader(name: string, value: string): void {
        this.headers.set(name, value)
    }

    /**
     * 获取请求RequestInit
     */
    public getRequestInit(): RequestInit {
        return this.requestInit
    }


    /**
     * 设置请求RequestInit
     * @param init 
     */
    public setRequestInit(init: RequestInit): void {
        if (init.body) {
            this.requestInit.body = init.body;
        }

        if (init.cache) {
            this.requestInit.cache = init.cache;
        }

        if (init.credentials) {
            this.requestInit.credentials = init.credentials;
        }

        if (init.headers) {
            this.requestInit.headers = init.headers;
        }

        if (init.integrity) {
            this.requestInit.integrity = init.integrity;
        }

        if (init.keepalive) {
            this.requestInit.keepalive = init.keepalive;
        }

        if (init.method) {
            this.requestInit.method = init.method;
        }

        if (init.mode) {
            this.requestInit.mode = init.mode;
        }

        if (init.redirect) {
            this.requestInit.redirect = init.redirect;
        }

        if (init.referrer) {
            this.requestInit.referrer = init.referrer;
        }

        if (init.referrerPolicy) {
            this.requestInit.referrerPolicy = init.referrerPolicy;
        }

        if (init.signal) {
            this.requestInit.signal = init.signal;
        }

        if (init.window) {
            this.requestInit.window = init.window;
        }
    }

    public get(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "GET";
        return this.ajax(url, this.getRequestInit());
    }

    public post(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "POST";
        return this.ajax(url, this.getRequestInit());
    }

    public put(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "PUT";
        return this.ajax(url, this.getRequestInit());
    }

    public patch(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "PATCH";
        return this.ajax(url, this.getRequestInit());
    }

    public delete(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "DELETE";
        return this.ajax(url, this.getRequestInit());

    }

    public head(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "HEAD";
        return this.ajax(url, this.getRequestInit());
    }

    public options(url: string, init?: RequestInit): Promise<Response> {
        if (init) {
            this.requestInit = init;
        }
        this.requestInit.method = "OPTIONS";
        return this.ajax(url, this.getRequestInit());
    }

    /**
     * AJAX 请求方法
     * @param url 
     * @param init 
     */
    public ajax(url: string, init: RequestInit): Promise<Response> {
        let request = new Request(url, init);
        // 调用request钩子函数,如果添加的钩子函数返回值是true,后续的钩子函数就不调用了,直接返回
        let callRequestHookResult = this.callRequestHook(request);
        if (callRequestHookResult.isBreak) {
            return new Promise<Response>(() => {
                return new Response();
            });
        }

        return fetch(callRequestHookResult.request)
            .then((response) => {
                // let response = response.clone();
                // 调用response钩子函数,如果添加的钩子函数返回值是true,后续的钩子函数就不调用了,直接返回
                let callResponseHookResult = this.callResponseHook(response);
                return callResponseHookResult.response;
            });
    }


    /**
     * 添加实例Request钩子函数
     * @param hookFunction 
     */
    public addRequestHook(hookFunction: requestHook) {
        this.requestHooks.push(hookFunction);
    }

    /**
     * 添加全局Request钩子函数
     * @param hookFunction 
     */
    public addGlobalRequestHook(hookFunction: requestHook) {
        Xhttp.globalRequestHooks.push(hookFunction);
    }

    /**
     * 添加实例Response钩子函数
     * @param hookFunction
     */
    public addResponseHook(hookFunction: responseHook) {
        this.responseHooks.push(hookFunction);
    }

    /**
     * 添加全局Response钩子函数
     * @param hookFunction 
     */
    public addGlobalResponseHook(hookFunction: responseHook) {
        Xhttp.globalResponseHooks.push(hookFunction);
    }
}


function exportDefault() {
    return new Xhttp();
}

export default exportDefault();

export {
    Xhttp
}



//===============================================================================================
/**
let token = "";

function downloadFile(path: string, filename: string) {
    let getHeaders = new Headers();
    // getHeaders.append('Content-Type', 'image/jpeg');
    getHeaders.append("token", token);
    let getInit: RequestInit = {
        method: "GET",
        headers: getHeaders,
        mode: "cors",
        cache: "default",
        credentials: "include",
    };

    let getRequest = new Request("/api/file/get?path=" + path, getInit);

    fetch(getRequest).then((res) =>
        res.blob().then((blob) => {
            var a = document.createElement("a");
            var url = window.URL.createObjectURL(blob);
            a.href = url;
            a.download = filename;
            a.click();
            window.URL.revokeObjectURL(url);
        })
    );
}

 * 下载文件(只能是文件,不能是目录)
 */

/**
 * 删除文件或目录

function deleteFile(index: Number, row: any) {
    console.log("删除文件");
    console.log(index);
    console.log(row);

    let formData = new FormData();
    formData.append("path", row.path);

    let getHeaders = new Headers();
    getHeaders.append("token", token);
    let getInit: RequestInit = {
        method: "POST",
        headers: getHeaders,
        mode: "cors",
        body: formData,
        cache: "default",
        credentials: "include",
    };

    let getRequest = new Request("/api/file/delete", getInit);
    fetch(getRequest)
        .then((res) => res.json())
        .then((json) => {
            console.log(json);
            // data.files = json.files;
        });
}

 */