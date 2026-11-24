import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import assert from 'node:assert/strict';
import test, { after } from 'node:test';
import ts from 'typescript';
const require = createRequire(import.meta.url);
const dir = mkdtempSync(join(tmpdir(), 'retrocast-web-test-'));
const cleanup = () => rmSync(dir, {recursive:true,force:true});
after(cleanup);
process.on('exit', cleanup);
try {
const storage = new Map();
globalThis.localStorage = {getItem:k=>storage.get(k)??null,setItem:(k,v)=>storage.set(k,v),removeItem:k=>storage.delete(k)};
function compile(source, name) {
 const file=join(dir,name+'.mjs');
 writeFileSync(file, ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.ES2022,jsx:ts.JsxEmit.ReactJSX}}).outputText);
 return pathToFileURL(file).href;
}
const apiUrl=compile(readFileSync(resolve('src/lib/api.ts'),'utf8'),'api');
const {api,ApiClientError}=await import(apiUrl);
const authSource=readFileSync(resolve('src/stores/auth.ts'),'utf8').replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"zustand"',JSON.stringify(pathToFileURL(require.resolve('zustand')).href));
const {useAuthStore}=await import(compile(authSource,'auth'));
test('204 deletes succeed, success and upload unwrap, errors preserve detail',async()=>{
 globalThis.fetch=async()=>new Response(null,{status:204});assert.equal(await api.delete('/empty'),undefined);
 globalThis.fetch=async()=>Response.json({data:{id:'9007199254740993'}});assert.equal((await api.get('/resource')).id,'9007199254740993');
 assert.equal((await api.upload('/file',new File(['a'],'a.txt'))).id,'9007199254740993');
 globalThis.fetch=async()=>Response.json({error:{code:'FORBIDDEN',message:'Role is too high'}},{status:403});
 await assert.rejects(api.patch('/role',{}),err=>err instanceof ApiClientError && err.message==='Role is too high');
});
test('refresh persists rotation twice and auth 401 does not recursively refresh',async()=>{
 useAuthStore.setState({refreshToken:'old'});let calls=0;
 globalThis.fetch=async()=>{calls++;return Response.json({data:{access_token:'access'+calls,refresh_token:'rotated'+calls}})};
 await useAuthStore.getState().refreshAccessToken();await useAuthStore.getState().refreshAccessToken();
 assert.equal(calls,2);assert.equal(storage.get('refreshToken'),'rotated2');assert.equal(useAuthStore.getState().refreshToken,'rotated2');
 calls=0;globalThis.fetch=async()=>{calls++;return Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401})};
 await assert.rejects(useAuthStore.getState().refreshAccessToken());assert.equal(calls,1);
});
test('logout revokes stored refresh token before clearing state',async()=>{
 useAuthStore.setState({refreshToken:'logout-token',accessToken:'access',isAuthenticated:true});
 let calls=0;globalThis.fetch=async(url,options)=>{calls++;assert.ok(url.endsWith('/api/v1/auth/logout'));assert.equal(JSON.parse(options.body).refresh_token,'logout-token');assert.equal(useAuthStore.getState().refreshToken,'logout-token');return new Response(null,{status:204})};
 await useAuthStore.getState().logout();assert.equal(calls,1);assert.equal(useAuthStore.getState().isAuthenticated,false);assert.equal(storage.get('refreshToken'),undefined);
});
test('Administrator permission remains unsigned in payload',()=>{
 const source=readFileSync(resolve('src/components/RoleEditor.tsx'),'utf8');
 const bit=Function('return '+source.match(/name: "Administrator", bit: (.*), category/)[1])();
 const expression=source.match(/setPermissions\(\(prev\) =>\s*([\s\S]*?),\s*\);/)[1];
 const toggle=Function('prev','bit','return '+expression);
 const permissions=toggle(0,bit);assert.equal(String(permissions),'2147483648');assert.equal(toggle(permissions,bit),0);
});

test('REST and gateway channel create converge without duplicate rows',async()=>{
 const source=readFileSync(resolve('src/stores/channels.ts'),'utf8').replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"zustand"',JSON.stringify(pathToFileURL(require.resolve('zustand')).href));
 const {useChannelsStore}=await import(compile(source,'channels'));
 const channel={id:'9007199254740993',guild_id:'guild',name:'test',type:0,position:0};
 useChannelsStore.getState().addChannel(channel);useChannelsStore.getState().addChannel(channel);
 assert.equal(useChannelsStore.getState().channelsByGuild.get('guild').length,1);
});

test('logout with expired access revokes the rotated refresh token',async()=>{
 useAuthStore.setState({refreshToken:'expired-access-refresh',accessToken:'expired',isAuthenticated:true});let calls=0;
 globalThis.fetch=async(url,options)=>{
  calls++;
  if(calls===1) return Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401});
  if(calls===2) {assert.ok(url.endsWith('/auth/refresh'));return Response.json({data:{access_token:'new-access',refresh_token:'current-refresh'}})}
  assert.ok(url.endsWith('/auth/logout'));assert.equal(JSON.parse(options.body).refresh_token,'current-refresh');return new Response(null,{status:204});
 };
 await useAuthStore.getState().logout();assert.equal(calls,3);assert.equal(useAuthStore.getState().isAuthenticated,false);
});

function setSignedInFixture() {
 storage.set('refreshToken','fixture-refresh');
 useAuthStore.setState({user:{id:'fixture-user'},refreshToken:'fixture-refresh',accessToken:'fixture-access',isAuthenticated:true});
}
function assertSignedOut() {
 const state=useAuthStore.getState();
 assert.equal(state.user,null);assert.equal(state.accessToken,null);assert.equal(state.refreshToken,null);
 assert.equal(state.isAuthenticated,false);assert.equal(storage.get('refreshToken'),undefined);
}
test('logout clears state and storage when fetch rejects',async()=>{
 setSignedInFixture();let calls=0;
 globalThis.fetch=async()=>{calls++;throw new TypeError('network unavailable')};
 await useAuthStore.getState().logout();assert.equal(calls,1);assertSignedOut();
});
test('logout clears state and storage when refresh fails with 401',async()=>{
 setSignedInFixture();const paths=[];
 globalThis.fetch=async(url)=>{paths.push(url);return Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401})};
 await useAuthStore.getState().logout();assert.equal(paths.length,2);assert.ok(paths[0].endsWith('/auth/logout'));assert.ok(paths[1].endsWith('/auth/refresh'));assertSignedOut();
});
test('logout clears state on 429, 500 and failed revoke retry',async()=>{
 for(const status of [429,500]) {
  setSignedInFixture();globalThis.fetch=async()=>Response.json({error:{code:'FAILED',message:'unavailable'}},{status});
  await useAuthStore.getState().logout();assertSignedOut();
 }
 setSignedInFixture();let calls=0;
 globalThis.fetch=async()=>{calls++;if(calls===2)return Response.json({data:{access_token:'rotated-access',refresh_token:'rotated-refresh'}});return Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401})};
 await useAuthStore.getState().logout();assert.equal(calls,3);assertSignedOut();
});

// Render the real component functions with deterministic hooks and inspect their
// event handlers/selectors. No source-expression matching for these regressions.
const jsxUrl=compile('export const jsx=(type,props,key)=>({type,props,key});export const jsxs=jsx;export const Fragment=Symbol("Fragment");','jsx');
const hooksUrl=compile(`export let values=[];let cursor=0;
 export function reset(next=[]){values=next;cursor=0}
 export function useState(initial){const i=cursor++;if(!(i in values))values[i]=initial;return [values[i],v=>{values[i]=typeof v==='function'?v(values[i]):v}]}
 export function useEffect(){};`,'hooks');
const hooks=await import(hooksUrl);
const emptyComponentUrl=compile('export default function Component(){return null};','empty-component');
function storeModule(name, state) {
 const url=compile('export let state;export const selected=[];export function setState(value){state=value;selected.length=0};export function '+name+'(selector){const value=selector(state);selected.push(value);return value};'+name+'.getState=()=>state;',name);
 return import(url).then(module=>{module.setState(state);return {url,module}});
}
async function component(file, replacements, suffix='') {
 let source=readFileSync(resolve('src/components/'+file+'.tsx'),'utf8')+suffix;
 source=source.replace(/from "([^"]+)"/g,(whole,name)=>{
  const target=name==='react'?hooksUrl:replacements[name]??(name.startsWith('@/components/')?emptyComponentUrl:undefined);
  return target?'from '+JSON.stringify(target):whole;
 });
 // The JSX runtime import is introduced by TypeScript after import rewriting.
 const url=compile(source,file+'-component');const path=new URL(url);
 writeFileSync(path,readFileSync(path,'utf8').replace('"react/jsx-runtime"',JSON.stringify(jsxUrl)));
 return import(url);
}
function nodes(tree) {
 if(!tree || typeof tree!=='object')return [];
 if(Array.isArray(tree))return tree.flatMap(nodes);
 return [tree,...nodes(tree.props?.children)];
}
function element(tree, predicate) {const match=nodes(tree).find(predicate);assert.ok(match,'component element missing');return match;}
test('invite revoke uses the versioned API path',async()=>{
 let requested;
 const fakeApi=compile('export const api={get:async()=>[],delete:async(path)=>{globalThis.inviteRevokePath=path}}','invite-api');
 const {default:InviteModal}=await component('InviteModal',{'@/lib/api':fakeApi});
 hooks.reset([[{code:'fixture-code',uses:0,max_uses:0}],false,false,null]);
 const tree=InviteModal({guildId:'guild',onClose(){}});
 await element(tree,n=>n.type==='button'&&n.props.children==='Revoke').props.onClick();
 requested=globalThis.inviteRevokePath;delete globalThis.inviteRevokePath;
 assert.equal(requested,'/api/v1/invites/fixture-code');assert.deepEqual(hooks.values[0],[]);
});
test('invite join selects the returned id and fetches its channels',async()=>{
 const selected=[],fetched=[];
 const guilds=await storeModule('useGuildsStore',{fetchGuilds:async()=>{},selectGuild:id=>selected.push(id)});
 const channels=await storeModule('useChannelsStore',{fetchChannels:async id=>fetched.push(id)});
 const dms=await storeModule('useDMsStore',{});
 const fakeApi=compile('export const api={post:async()=>({id:"joined-guild"})}','join-api');
 const {JoinGuildModal}=await component('ServerList',{'@/lib/api':fakeApi,'@/stores/guilds':guilds.url,'@/stores/channels':channels.url,'@/stores/dms':dms.url},'\nexport {JoinGuildModal};');
 hooks.reset(['fixture-code',false,'']);
 const tree=JoinGuildModal({onClose(){}});
 await element(tree,n=>n.type==='form').props.onSubmit({preventDefault(){}});
 assert.deepEqual(selected,['joined-guild']);assert.deepEqual(fetched,['joined-guild']);
});
test('MemberList empty selectors keep the same snapshot on repeated reads',async()=>{
 const members=await storeModule('useMembersStore',{membersByGuild:new Map(),fetchMembers:async()=>{}});
 const roles=await storeModule('useRolesStore',{rolesByGuild:new Map(),fetchRoles:async()=>{}});
 const presence=await storeModule('usePresenceStore',{presences:new Map()});
 const users=await storeModule('useUsersStore',{users:new Map()});
 const {default:MemberList}=await component('MemberList',{'@/stores/members':members.url,'@/stores/roles':roles.url,'@/stores/presence':presence.url,'@/stores/users':users.url});
 hooks.reset();MemberList({guildId:'empty'});const firstMembers=members.module.selected[0],firstRoles=roles.module.selected[0];
 members.module.selected.length=0;roles.module.selected.length=0;
 hooks.reset();MemberList({guildId:'empty'});
 assert.equal(members.module.selected[0],firstMembers);assert.equal(roles.module.selected[0],firstRoles);
});

test('UserSettings always navigates when disconnect or logout throws',async()=>{
 let logoutCalls=0;
 const auth=await storeModule('useAuthStore',{user:{id:'fixture-user',display_name:'Fixture'},logout:async()=>{logoutCalls++;throw new Error('logout fixture failure')}});
 const fakeApi=compile('export const api={};','settings-api');
 const fakeGateway=compile('export let fail=false;export function setFail(value){fail=value};export const gateway={disconnect(){if(fail)throw new Error("disconnect fixture failure")}};','settings-gateway');
 const gatewayMock=await import(fakeGateway);
 const {default:UserSettingsModal}=await component('UserSettingsModal',{'@/stores/auth':auth.url,'@/lib/api':fakeApi,'@/lib/gateway':fakeGateway});
 const previousWindow=globalThis.window;
 try {
  for(const fail of [false,true]) {
   globalThis.window={location:{href:'/app'}};gatewayMock.setFail(fail);hooks.reset();
   const tree=UserSettingsModal({onClose(){}});
   await element(tree,n=>n.type==='button'&&n.props.children==='Log Out').props.onClick();
   assert.equal(globalThis.window.location.href,'/server');
   assert.equal(logoutCalls,fail?2:1);
  }
 } finally {globalThis.window=previousWindow}
});

} catch (error) {
 cleanup();
 throw error;
}
