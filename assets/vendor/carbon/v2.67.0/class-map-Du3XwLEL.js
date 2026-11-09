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
import{a as e}from"./decorate-BgofJUJ5.js";import{n as t,r as n,t as r}from"./directive-uL8lMFH7.js";
/**
* @license
* Copyright 2018 Google LLC
* SPDX-License-Identifier: BSD-3-Clause
*/const i=r(class extends t{constructor(e){if(super(e),e.type!==n.ATTRIBUTE||e.name!==`class`||e.strings?.length>2)throw Error("`classMap()` can only be used in the `class` attribute and must be the only part in the attribute.")}render(e){return` `+Object.keys(e).filter(t=>e[t]).join(` `)+` `}update(t,[n]){if(this.st===void 0){this.st=new Set,t.strings!==void 0&&(this.nt=new Set(t.strings.join(` `).split(/\s/).filter(e=>e!==``)));for(let e in n)n[e]&&!this.nt?.has(e)&&this.st.add(e);return this.render(n)}let r=t.element.classList;for(let e of this.st)e in n||(r.remove(e),this.st.delete(e));for(let e in n){let t=!!n[e];t===this.st.has(e)||this.nt?.has(e)||(t?(r.add(e),this.st.add(e)):(r.remove(e),this.st.delete(e)))}return e}});export{i as t};