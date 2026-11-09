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
import{d as e,u as t}from"./decorate-BgofJUJ5.js";
/**
* @license
* Copyright 2017 Google LLC
* SPDX-License-Identifier: BSD-3-Clause
*/const n={attribute:!0,type:String,converter:e,reflect:!1,hasChanged:t},r=(e=n,t,r)=>{let{kind:i,metadata:a}=r,o=globalThis.litPropertyMetadata.get(a);if(o===void 0&&globalThis.litPropertyMetadata.set(a,o=new Map),i===`setter`&&((e=Object.create(e)).wrapped=!0),o.set(r.name,e),i===`accessor`){let{name:n}=r;return{set(r){let i=t.get.call(this);t.set.call(this,r),this.requestUpdate(n,i,e,!0,r)},init(t){return t!==void 0&&this.C(n,void 0,e,t),t}}}if(i===`setter`){let{name:n}=r;return function(r){let i=this[n];t.call(this,r),this.requestUpdate(n,i,e,!0,r)}}throw Error(`Unsupported decorator location: `+i)};function i(e){return(t,n)=>typeof n==`object`?r(e,t,n):((e,t,n)=>{let r=t.hasOwnProperty(n);return t.constructor.createProperty(n,e),r?Object.getOwnPropertyDescriptor(t,n):void 0})(e,t,n)}
/**
* @license
* Copyright 2017 Google LLC
* SPDX-License-Identifier: BSD-3-Clause
*/function a(e){return i({...e,state:!0,attribute:!1})}export{i as n,a as t};