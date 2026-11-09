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
import{n as e,p as t,s as n,t as r}from"./decorate-BgofJUJ5.js";import{n as i}from"./decorators-lcU6xUBg.js";import{t as a}from"./query-BCEhJtix.js";import"./settings-CZDgQBsm.js";import{t as o}from"./carbon-element-DgPTymXo.js";import{t as s}from"./class-map-Du3XwLEL.js";import{t as c}from"./focus-BAk8V28j.js";import{t as l}from"./if-defined-Dnjx5V1i.js";var u=t([`.cds--link{box-sizing:border-box;padding:0;border:0;margin:0;font-family:inherit;font-size:100%;vertical-align:baseline}.cds--link *,.cds--link :after,.cds--link :before{box-sizing:inherit}.cds--link{font-size:var(--cds-body-compact-01-font-size,.875rem);font-weight:var(--cds-body-compact-01-font-weight,400);line-height:var(--cds-body-compact-01-line-height,1.28572);letter-spacing:var(--cds-body-compact-01-letter-spacing,.16px);display:inline-flex;color:var(--cds-link-text-color,var(--cds-link-primary,#0f62fe));outline:none;text-decoration:none;transition:color 70ms cubic-bezier(.2,0,.38,.9)}@media (any-hover:hover){.cds--link:hover{color:var(--cds-link-hover-text-color,var(--cds-link-primary-hover,#0043ce));text-decoration:underline}}.cds--link:active:not(.cds--link--disabled),.cds--link:active:visited{outline:1px solid var(--cds-focus,#0f62fe)}@media screen and (prefers-contrast){.cds--link:active:not(.cds--link--disabled),.cds--link:active:visited{outline-style:dotted}}.cds--link:active:not(.cds--link--disabled),.cds--link:active:visited{color:var(--cds-link-text-color,var(--cds-link-primary,#0f62fe));outline-color:var(--cds-link-focus-text-color,var(--cds-focus,#0f62fe));text-decoration:underline}.cds--link:focus:not(.cds--link--disabled){outline:1px solid var(--cds-focus,#0f62fe)}@media screen and (prefers-contrast){.cds--link:focus:not(.cds--link--disabled){outline-style:dotted}}.cds--link:focus:not(.cds--link--disabled){outline-color:var(--cds-link-focus-text-color,var(--cds-focus,#0f62fe));text-decoration:underline}.cds--link:visited{color:var(--cds-link-text-color,var(--cds-link-primary,#0f62fe))}@media (any-hover:hover){.cds--link:visited:hover{color:var(--cds-link-hover-text-color,var(--cds-link-primary-hover,#0043ce))}.cds--link:active:not(.cds--link--disabled):visited:hover{color:var(--cds-link-text-color,var(--cds-link-primary,#0f62fe))}}.cds--link--disabled{box-sizing:border-box;padding:0;border:0;margin:0;font-family:inherit;font-size:100%;vertical-align:baseline}.cds--link--disabled *,.cds--link--disabled :after,.cds--link--disabled :before{box-sizing:inherit}.cds--link--disabled{font-size:var(--cds-body-compact-01-font-size,.875rem);font-weight:var(--cds-body-compact-01-font-weight,400);line-height:var(--cds-body-compact-01-line-height,1.28572);letter-spacing:var(--cds-body-compact-01-letter-spacing,.16px);color:var(--cds-text-disabled,rgba(22,22,22,.25));cursor:not-allowed;font-weight:400;text-decoration:none}@media (any-hover:hover){.cds--link--disabled:hover{color:var(--cds-text-disabled,rgba(22,22,22,.25));text-decoration:none}}.cds--link.cds--link--visited:visited{color:var(--cds-link-visited-text-color,var(--cds-link-visited,#8a3ffc))}@media (any-hover:hover){.cds--link.cds--link--visited:visited:hover{color:var(--cds-link-hover-text-color,var(--cds-link-primary-hover,#0043ce))}.cds--link.cds--link--visited:active:not(.cds--link--disabled):visited:hover{color:var(--cds-link-text-color,var(--cds-link-primary,#0f62fe))}}.cds--link.cds--link--inline{display:inline;text-decoration:underline}.cds--link--disabled.cds--link--inline{text-decoration:underline}.cds--link--sm{font-size:var(--cds-helper-text-01-font-size,.75rem);line-height:var(--cds-helper-text-01-line-height,1.33333);letter-spacing:var(--cds-helper-text-01-letter-spacing,.32px)}.cds--link--lg{font-size:var(--cds-body-compact-02-font-size,1rem);font-weight:var(--cds-body-compact-02-font-weight,400);line-height:var(--cds-body-compact-02-line-height,1.375);letter-spacing:var(--cds-body-compact-02-letter-spacing,0)}.cds--link--icon{display:inline-block}.cds--link__icon{display:inline-flex;align-items:center;block-size:1lh;line-height:inherit;margin-inline-start:.5rem;vertical-align:text-bottom}.cds--link--lg .cds--link__icon>svg{block-size:1.25rem;inline-size:1.25rem}.cds--link--lg .cds--link__icon{block-size:1.25rem}:host(cds-link){outline:none}:host(cds-link) .cds--link--disabled{color:var(--cds-text-disabled,rgba(22,22,22,.25))}:host(cds-link) .cds--link__icon[hidden]{display:none}:host(cds-link) .cds--link--lg .cds--link__icon ::slotted(svg){block-size:1.25rem;inline-size:1.25rem}`]);const d={MEDIUM:`md`,SMALL:`sm`,LARGE:`lg`};let f=class extends c(e){constructor(...e){super(...e),this._hasIcon=!1,this.disabled=!1,this.inline=!1,this.size=d.MEDIUM,this.visited=!1}_handleSlotChange({target:e}){let{name:t}=e,n=e.assignedNodes().some(e=>e.nodeType!==Node.TEXT_NODE||e.textContent?.trim());this[t===`icon`?`_hasIcon`:``]=n,this.requestUpdate()}get _classes(){let{disabled:e,size:t,inline:n,visited:r,_hasIcon:i}=this;return s({"cds--link":!0,"cds--link--disabled":e,"cds--link--icon":i,"cds--link--inline":n,[`cds--link--${t}`]:t,"cds--link--visited":r})}_renderInner(){let{_hasIcon:e,_handleSlotChange:t}=this;return n`
      <slot @slotchange="${t}"></slot>
      <span ?hidden="${!e}" class="${`cds`}--link__icon">
        <slot name="icon" @slotchange="${t}"></slot>
      </span>
    `}_renderDisabledLink(){let{_classes:e}=this;return n`
      <p id="link" part="link" class="${e}">${this._renderInner()}</p>
    `}_renderLink(){let{download:e,href:t,hreflang:r,linkRole:i,ping:a,rel:o,target:s,type:c,_classes:u}=this;return n`
      <a
        tabindex="0"
        id="link"
        role="${l(i)}"
        class="${u}"
        part="link"
        download="${l(e)}"
        href="${l(t)}"
        hreflang="${l(r)}"
        ping="${l(a)}"
        rel="${l(o)}"
        target="${l(s)}"
        type="${l(c)}">
        ${this._renderInner()}
      </a>
    `}render(){let{disabled:e}=this;return e?this._renderDisabledLink():this._renderLink()}static{this.shadowRootOptions={...e.shadowRootOptions,delegatesFocus:!0}}static{this.styles=u}};r([a(`#link`)],f.prototype,`_linkNode`,void 0),r([i({type:Boolean,reflect:!0})],f.prototype,`disabled`,void 0),r([i({reflect:!0})],f.prototype,`download`,void 0),r([i({reflect:!0})],f.prototype,`href`,void 0),r([i({reflect:!0})],f.prototype,`hreflang`,void 0),r([i({type:Boolean,reflect:!0})],f.prototype,`inline`,void 0),r([i({attribute:`link-role`})],f.prototype,`linkRole`,void 0),r([i({reflect:!0})],f.prototype,`ping`,void 0),r([i({reflect:!0})],f.prototype,`rel`,void 0),r([i({reflect:!0})],f.prototype,`size`,void 0),r([i({reflect:!0})],f.prototype,`target`,void 0),r([i({reflect:!0})],f.prototype,`type`,void 0),r([i({type:Boolean,reflect:!0})],f.prototype,`visited`,void 0),f=r([o(`cds-link`)],f);var p=f;export{p as t};