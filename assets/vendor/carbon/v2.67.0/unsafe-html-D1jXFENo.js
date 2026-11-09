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
import{a as e,r as t}from"./decorate-BgofJUJ5.js";import{n,r,t as i}from"./directive-uL8lMFH7.js";
/**
* @license
* Copyright 2017 Google LLC
* SPDX-License-Identifier: BSD-3-Clause
*/var a=class extends n{constructor(e){if(super(e),this.it=t,e.type!==r.CHILD)throw Error(this.constructor.directiveName+`() can only be used in child bindings`)}render(n){if(n===t||n==null)return this._t=void 0,this.it=n;if(n===e)return n;if(typeof n!=`string`)throw Error(this.constructor.directiveName+`() called with a non-string value`);if(n===this.it)return this._t;this.it=n;let r=[n];return r.raw=r,this._t={_$litType$:this.constructor.resultType,strings:r,values:[]}}};a.directiveName=`unsafeHTML`,a.resultType=1;const o=i(a);export{o as n,a as t};