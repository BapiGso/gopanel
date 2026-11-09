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
var e=class e{constructor(e){this._groups={},this.constructor._instances.set(e,this)}shouldBeFocusable(e){if(e.checked)return!0;let{name:t}=e,n=this._groups[t];return n&&Array.from(n).some(e=>e.checked)?!1:!n||n.size===1||this.getSortedGroup(e)[0]===e}getSortedGroup(e){let t=this._groups[e.name];return t&&Array.from(t).sort((e,t)=>{let n=e.compareDocumentPosition(t);return n&Node.DOCUMENT_POSITION_FOLLOWING||n&Node.DOCUMENT_POSITION_CONTAINED_BY?-1:n&Node.DOCUMENT_POSITION_PRECEDING||n&Node.DOCUMENT_POSITION_CONTAINS?1:0})}add(e){let{name:t}=e;if(t){let n=this._groups;n[t]||(n[t]=new Set),n[t].add(e)}return this}delete(e,t=e.name){let n=this._groups[t];return n?n.delete(e):!1}select(e,t){let n=this._groups[e.name];if(n){if(e.disabled)return;e.checked=!t||!0,e.tabIndex=0,e.focus(),n.forEach(n=>{e!==n&&(n.checked=t||!1,n.tabIndex=-1)})}}navigate(e,t){let n=this.getSortedGroup(e),r=n.length,i=n.indexOf(e);for(let e=0;e<r;e++){i=(i+t+r)%r;let e=n[i];if(!e.disabled)return e}return e}static{this._instances=new WeakMap}static get(t){return this._instances.get(t)||new e(t)}};export{e as t};