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
import{s as e,t}from"./decorate-BgofJUJ5.js";import"./settings-CZDgQBsm.js";import{t as n}from"./carbon-element-DgPTymXo.js";import{t as r}from"./class-map-Du3XwLEL.js";import{t as i}from"./if-defined-Dnjx5V1i.js";import{n as a,t as o}from"./button-DyPDZ5DV.js";let s=class extends o{_handleClickLinkSkeleton(e){this.disabled&&(e.preventDefault(),e.stopPropagation())}render(){let{autofocus:t,disabled:n,download:a,href:o,hreflang:s,ping:c,rel:l,size:u,target:d,type:f}=this,p=r({"cds--btn":!0,"cds--skeleton":!0,[`cds--btn--${u}`]:!!u,[`cds--layout--size-${u}`]:!!u});return o?e`
          <a
            id="button"
            role="button"
            class="${p}"
            download="${i(a)}"
            href="${i(o)}"
            hreflang="${i(s)}"
            ping="${i(c)}"
            rel="${i(l)}"
            target="${i(d)}"
            type="${i(f)}"
            @click="${this._handleClickLinkSkeleton}"></a>
        `:e`
          <button
            id="button"
            class="${p}"
            ?autofocus="${t}"
            ?disabled="${n}"
            type="${i(f)}"></button>
        `}static{this.styles=a}};s=t([n(`cds-button-skeleton`)],s);