import{c as u}from"./chunk-createLucideIcon-SgBSd_zz.js";import{R as e,j as f}from"./entry-index-Cq13IMyc.js";/**
 * @license lucide-react v0.511.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const m=[["path",{d:"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z",key:"1a8usu"}],["path",{d:"m15 5 4 4",key:"1mk7zo"}]],x=u("pencil",m),n=e.createContext(void 0),N=({children:s})=>{const[r,i]=e.useState([]),[c,o]=e.useState(!1),[d,l]=e.useState(null),a=()=>{fetch("/api/admin/client/list").then(t=>t.json()).then(t=>{i(t),o(!1)}).catch(t=>{l(t.message),o(!1)})};return e.useEffect(()=>{o(!0),a()},[]),f.jsx(n.Provider,{value:{nodeDetail:r,isLoading:c,error:d,refresh:a},children:s})},D=()=>{const s=e.useContext(n);if(s===void 0)throw new Error("useNodeDetails must be used within a NodeDetailsProvider");return s};export{N,x as P,D as u};
