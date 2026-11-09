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
import{t as e}from"./on-DEY8hdnf.js";const t=t=>{class n extends t{constructor(...e){super(...e),this._hFormdata=null}connectedCallback(){super.connectedCallback();let t=this.closest(`form`);t&&(this._hFormdata=e(t,`formdata`,this._handleFormdata.bind(this)))}disconnectedCallback(){this._hFormdata&&=this._hFormdata.release(),super.disconnectedCallback()}}return n};export{t};