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
import{n as e,s as t,t as n}from"./decorate-BgofJUJ5.js";import{n as r}from"./decorators-lcU6xUBg.js";import"./settings-CZDgQBsm.js";import{t as i}from"./carbon-element-DgPTymXo.js";import{t as a}from"./class-map-Du3XwLEL.js";import{n as o}from"./text-input-Bv2-AycR.js";let s=class extends e{constructor(...e){super(...e),this.hideLabel=!1}render(){let{hideLabel:e,size:n}=this,r=a({"cds--form-item":!0,[`cds--layout--size-${n}`]:n!==void 0});return t`
      <div class=${r}>
        ${e?``:t`<span class="${`cds`}--label ${`cds`}--skeleton"></span>`}
        <div class="${`cds`}--text-input ${`cds`}--skeleton"></div>
      </div>
    `}static{this.styles=o}};n([r({type:Boolean,reflect:!0,attribute:`hide-label`})],s.prototype,`hideLabel`,void 0),n([r({reflect:!0})],s.prototype,`size`,void 0),s=n([i(`cds-text-input-skeleton`)],s);var c=s;export{c as t};