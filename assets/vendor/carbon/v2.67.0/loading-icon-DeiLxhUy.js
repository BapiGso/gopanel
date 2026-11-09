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
import{s as e}from"./decorate-BgofJUJ5.js";import"./settings-CZDgQBsm.js";var t=({description:t,small:n})=>{let r=n?`42`:`44`;return e`
    <svg class="${`cds`}--loading__svg" viewBox="0 0 100 100">
      ${t?e` <title>${t}</title> `:void 0}
      <circle
        ?hidden="${!n}"
        class="${`cds`}--loading__background"
        cx="50%"
        cy="50%"
        r="${r}" />
      <circle
        class="${`cds`}--loading__stroke"
        cx="50%"
        cy="50%"
        r="${r}" />
    </svg>
  `};export{t};