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
import{t as e}from"./on-DEY8hdnf.js";const t=/^((document|window|parentRoot|shadowRoot):)?([\w-]+)$/,n=n=>{class r extends n{constructor(...e){super(...e),this._handles=new Set}connectedCallback(){super.connectedCallback();let n=this.constructor._hostListeners;Object.keys(n).forEach(r=>{Object.keys(n[r]).forEach(i=>{let a=t.exec(i);if(!a)throw Error(`Could not parse the event name: ${r}`);let[,,o,s]=a,c={document:this.ownerDocument,window:this.ownerDocument.defaultView,parentRoot:this.getRootNode(),shadowRoot:this.shadowRoot}[o]||this,{options:l}=n[r][i];this._handles.add(e(c,this.constructor[s]??s,this[r],l))})})}disconnectedCallback(){this._handles.forEach(e=>{e.release(),this._handles.delete(e)}),super.disconnectedCallback()}static{this._hostListeners={}}}return r};export{n as t};