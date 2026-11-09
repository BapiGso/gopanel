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
const e=e=>{class t extends e{_getValidityMessage(e){return{"":``,required:this.requiredValidityMessage}[e]}_testValidity(){let{required:e,value:t}=this;return e&&!t?`required`:``}checkValidity(){let e=this._testValidity();return e===``?(this.invalid=!1,this.validityMessage=``,!0):(this.dispatchEvent(new CustomEvent(`invalid`,{bubbles:!1,cancelable:!0,composed:!1}))&&(this.invalid=!0,this.validityMessage=this._getValidityMessage(e)),!1)}setCustomValidity(e){this.invalid=!!e,this.validityMessage=e}}return t};export{e as t};