/**
* @license
*
* Copyright IBM Corp. 2019, 2026
*
* This source code is licensed under the Apache-2.0 license found in the
* LICENSE file in the root directory of this source tree.
*
* Also refer to the following links for third-party dependencies:
* https://www.npmjs.com/package/@floating-ui/core
* https://www.npmjs.com/package/@floating-ui/dom
* https://www.npmjs.com/package/@floating-ui/utils
* https://www.npmjs.com/package/@ibm/telemetry-js
* https://www.npmjs.com/package/@lit/context
* https://www.npmjs.com/package/@lit/reactive-element
* https://www.npmjs.com/package/lit
* https://www.npmjs.com/package/lit-element
* https://www.npmjs.com/package/lit-html
* https://www.npmjs.com/package/lodash-es
* https://www.npmjs.com/package/tslib
*/
const e=(e,t,n,r)=>{let i=n._hostListeners;if(!i)throw Error("The method `@HostListener()` is defined on has to be of a class that has `HostListerMixin`.");i[r]||(i[r]={}),i[r][e]={options:t}},t=(t,n,r)=>{let{kind:i,key:a,placement:o}=r;if(!(i===`method`&&o===`prototype`||i===`field`&&o===`own`))throw Error("`@HostListener()` must be defined on instance methods, but you may have defined it on static, field, etc.");return{...r,finisher(r){e(t,n,r,a)}}},n=(n,r)=>(i,a)=>a===void 0?t(n,r,i):e(n,r,i.constructor,a);export{n as t};