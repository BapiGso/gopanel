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
import{f as e,n as t,s as n,t as r}from"./decorate-BgofJUJ5.js";import{n as i,t as a}from"./decorators-lcU6xUBg.js";import"./settings-CZDgQBsm.js";import{t as o}from"./carbon-element-DgPTymXo.js";import{n as s,r as c}from"./deep-shadow-contains-Mbh9LOw4.js";import{t as l}from"./defs-tiELxKm5.js";import{t as u}from"./tooltip-DRAfD1--.js";import"./popover.min.js";let d=class extends s(t){constructor(...e){super(...e),this.align=`bottom`,this.defaultOpen=!1,this.openOnHover=!1,this.open=!1,this._tooltipId=`cds--definition-tooltip-${Math.random().toString(16).slice(2)}`}connectedCallback(){super.connectedCallback(),e(this.renderRoot,[c,u]),this.hasAttribute(`default-open`)&&(this.open=!0)}_handleBlur(){this.open=!1}_handleMouseDown(){this.open=!this.open}_handleKeyDown(e){let{key:t}=e;t===`Esc`||t===`Escape`?this.open&&=(e.stopPropagation(),!1):(t===` `||t===`Enter`)&&(e.preventDefault(),this.open=!this.open)}_handleMouseEnter(){this.openOnHover&&(this.open=!0)}_handleMouseLeave(){this.open=!1}_handleFocus(){this.open=!0}render(){let{align:e,open:t,_tooltipId:r}=this;return n`
      <cds-popover
        @mouseenter=${this._handleMouseEnter}
        @mouseleave=${this._handleMouseLeave}
        highContrast
        .autoalign=${this.autoalign}
        .dropShadow=${!1}
        align=${e}
        .open=${t}>
        <button
          @focus=${this._handleFocus}
          @blur=${this._handleBlur}
          @mousedown=${this._handleMouseDown}
          @keydown=${this._handleKeyDown}
          aria-controls=${r}
          aria-describedby=${r}
          aria-expanded=${t}
          part="definition-term"
          class="${`cds`}--definition-term">
          <slot></slot>
        </button>
        <cds-popover-content id=${r}>
          <slot name="definition"></slot>
        </cds-popover-content>
      </cds-popover>
    `}static{this.styles=u}};r([i({reflect:!0,type:l})],d.prototype,`align`,void 0),r([i({type:Boolean,reflect:!0})],d.prototype,`autoalign`,void 0),r([i({type:Boolean,reflect:!0,attribute:`default-open`})],d.prototype,`defaultOpen`,void 0),r([i({reflect:!0,type:Boolean,attribute:`open-on-hover`})],d.prototype,`openOnHover`,void 0),r([a()],d.prototype,`open`,void 0),d=r([o(`cds-definition-tooltip`)],d);