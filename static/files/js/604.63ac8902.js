"use strict";(self["webpackChunk_radiantearth_stac_browser"]=self["webpackChunk_radiantearth_stac_browser"]||[]).push([[604],{2902:function(t,r,e){e.d(r,{Z:function(){return u}});var n=function(){var t=this,r=t._self._c;return Array.isArray(t.roles)?r("div",{staticClass:"roles badges"},t._l(t.roles,(function(e){return r("b-badge",{key:e,attrs:{variant:"secondary"}},[t._v(" "+t._s(t.displayRole(e))+" ")])})),1):t._e()},i=[],o={name:"ProviderRoles",props:{roles:{type:Array,default:null}},methods:{displayRole(t){let r=`providers.role.${t}`;return this.$te(r)?this.$t(r):t}}},s=o,a=e(1001),l=(0,a.Z)(s,n,i,!1,null,"3f311e56",null),u=l.exports},30604:function(t,r,e){e.r(r),e.d(r,{default:function(){return f}});var n=function(){var t=this,r=t._self._c;return r("section",{staticClass:"providers mb-4"},[r("h2",[t._v(t._s(t.$tc("providers.title",t.count)))]),t.isSimple?r("b-list-group",{staticClass:"mimic-expandable-card"},t._l(t.providers,(function(e,n){return r("b-list-group-item",{key:n,staticClass:"provider",attrs:{href:e.url,disabled:!e.url,target:"_blank",variant:"provider"}},[r("span",{staticClass:"title"},[t._v(t._s(e.name))]),r("ProviderRoles",{attrs:{roles:e.roles}})],1)})),1):r("div",{staticClass:"accordion",attrs:{role:"tablist"}},t._l(t.providers,(function(t,e){return r("Provider",{key:e,attrs:{id:String(e),provider:t}})})),1)],1)},i=[],o=e(70322),s=e(88367),a=e(2902),l=e(79879),u={name:"Providers",components:{BListGroup:o.N,BListGroupItem:s.f,Provider:()=>e.e(3135).then(e.bind(e,53135)),ProviderRoles:a.Z},props:{providers:{type:Array,required:!0}},computed:{count(){return l.ZP.size(this.providers)},isSimple(){return!this.providers.find((t=>{const r=["url","name","roles"];return Object.keys(t).filter((t=>!r.includes(t))).length>0}))}}},c=u,p=e(1001),d=(0,p.Z)(c,n,i,!1,null,null,null),f=d.exports},88367:function(t,r,e){e.d(r,{f:function(){return m}});var n=e(1915),i=e(69558),o=e(94689),s=e(12299),a=e(11572),l=e(26410),u=e(67040),c=e(20451),p=e(30488),d=e(67347);function f(t,r){var e=Object.keys(t);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(t);r&&(n=n.filter((function(r){return Object.getOwnPropertyDescriptor(t,r).enumerable}))),e.push.apply(e,n)}return e}function b(t){for(var r=1;r<arguments.length;r++){var e=null!=arguments[r]?arguments[r]:{};r%2?f(Object(e),!0).forEach((function(r){v(t,r,e[r])})):Object.getOwnPropertyDescriptors?Object.defineProperties(t,Object.getOwnPropertyDescriptors(e)):f(Object(e)).forEach((function(r){Object.defineProperty(t,r,Object.getOwnPropertyDescriptor(e,r))}))}return t}function v(t,r,e){return r in t?Object.defineProperty(t,r,{value:e,enumerable:!0,configurable:!0,writable:!0}):t[r]=e,t}var h=["a","router-link","button","b-link"],g=(0,u.CE)(d.NQ,["event","routerTag"]);delete g.href.default,delete g.to.default;var y=(0,c.y2)((0,u.GE)(b(b({},g),{},{action:(0,c.pi)(s.U5,!1),button:(0,c.pi)(s.U5,!1),tag:(0,c.pi)(s.N0,"div"),variant:(0,c.pi)(s.N0)})),o.KT),m=(0,n.l7)({name:o.KT,functional:!0,props:y,render:function(t,r){var e,n=r.props,o=r.data,s=r.children,u=n.button,f=n.variant,b=n.active,y=n.disabled,m=(0,p.u$)(n),_=u?"button":m?d.we:n.tag,O=!!(n.action||m||u||(0,a.kI)(h,n.tag)),P={},j={};return(0,l.YR)(_,"button")?(o.attrs&&o.attrs.type||(P.type="button"),n.disabled&&(P.disabled=!0)):j=(0,c.uj)(g,n),t(_,(0,i.b)(o,{attrs:P,props:j,staticClass:"list-group-item",class:(e={},v(e,"list-group-item-".concat(f),f),v(e,"list-group-item-action",O),v(e,"active",b),v(e,"disabled",y),e)}),s)}})},70322:function(t,r,e){e.d(r,{N:function(){return p}});var n=e(1915),i=e(69558),o=e(94689),s=e(12299),a=e(33284),l=e(20451);function u(t,r,e){return r in t?Object.defineProperty(t,r,{value:e,enumerable:!0,configurable:!0,writable:!0}):t[r]=e,t}var c=(0,l.y2)({flush:(0,l.pi)(s.U5,!1),horizontal:(0,l.pi)(s.gL,!1),tag:(0,l.pi)(s.N0,"div")},o.DX),p=(0,n.l7)({name:o.DX,functional:!0,props:c,render:function(t,r){var e=r.props,n=r.data,o=r.children,s=""===e.horizontal||e.horizontal;s=!e.flush&&s;var l={staticClass:"list-group",class:u({"list-group-flush":e.flush,"list-group-horizontal":!0===s},"list-group-horizontal-".concat(s),(0,a.HD)(s))};return t(e.tag,(0,i.b)(n,l),o)}})}}]);
//# sourceMappingURL=604.63ac8902.js.map