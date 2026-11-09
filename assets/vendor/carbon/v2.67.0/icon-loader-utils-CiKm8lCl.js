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
import{t as e}from"./directive-uL8lMFH7.js";import{t}from"./unsafe-html-D1jXFENo.js";
/**
* @license
* Copyright 2017 Google LLC
* SPDX-License-Identifier: BSD-3-Clause
*/var n=class extends t{};n.directiveName=`unsafeSVG`,n.resultType=2;const r=e(n);function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter(function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable})),n.push.apply(n,r)}return n}function a(e){for(var t=1;t<arguments.length;t++){var n=arguments[t]==null?{}:arguments[t];t%2?i(Object(n),!0).forEach(function(t){o(e,t,n[t])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach(function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))})}return e}function o(e,t,n){return t=u(t),t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function s(e,t){if(e==null)return{};for(var n={},r=Object.keys(e),i,a=0;a<r.length;a++)i=r[a],!(t.indexOf(i)>=0)&&(n[i]=e[i]);return n}function c(e,t){if(e==null)return{};var n=s(e,t),r,i;if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(i=0;i<a.length;i++)r=a[i],!(t.indexOf(r)>=0)&&Object.prototype.propertyIsEnumerable.call(e,r)&&(n[r]=e[r])}return n}function l(e,t){if(typeof e!=`object`||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var r=n.call(e,t||`default`);if(typeof r!=`object`)return r;throw TypeError(`@@toPrimitive must return a primitive value.`)}return(t===`string`?String:Number)(e)}function u(e){var t=l(e,`string`);return typeof t==`symbol`?t:String(t)}var d=[`width`,`height`,`viewBox`],f=[`tabindex`],p={focusable:`false`,preserveAspectRatio:`xMidYMid meet`};function m(){var e=arguments.length>0&&arguments[0]!==void 0?arguments[0]:{},t=e.width,n=e.height,r=e.viewBox,i=r===void 0?`0 0 ${t} ${n}`:r,o=c(e,d),s=o.tabindex,l=c(o,f),u=a(a(a({},p),l),{},{width:t,height:n,viewBox:i});return u[`aria-label`]||u[`aria-labelledby`]||u.title?(u.role=`img`,s!=null&&(u.focusable=`true`,u.tabindex=s)):u[`aria-hidden`]=!0,u}function h(e){return Object.keys(e).reduce(function(t,n,r){var i=`${n}="${e[n]}"`;return r===0?i:t+` `+i},``)}function g(e){return`default`in e&&e.default?e.default:e}function _(e,t={}){let n=g(e);return n.attrs||={},`<svg ${h(m({...n.attrs,...t}))}>${(n.content||[]).map(e=>typeof e==`string`?e:v(e)).join(``)}</svg>`}function v(e){if(typeof e==`string`)return e;let{elem:t=`svg`,attrs:n={},content:r=[]}=e,i=r.map(v).join(``);return`<${t} ${h(n)}>${i}</${t}>`}function y(e){return(t={})=>{let n=_(e,t);return r(n)}}export{r as n,y as t};