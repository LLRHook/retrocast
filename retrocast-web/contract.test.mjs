import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import assert from 'node:assert/strict';
import test, { after } from 'node:test';
import ts from 'typescript';
const require = createRequire(import.meta.url);
const dir = mkdtempSync(join(tmpdir(), 'l44-retrocast-web-test-'));
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
 storage.set('refreshToken','old');useAuthStore.setState({refreshToken:'old'});let calls=0;
 globalThis.fetch=async()=>{calls++;return Response.json({data:{access_token:'access'+calls,refresh_token:'rotated'+calls}})};
 await useAuthStore.getState().refreshAccessToken();await useAuthStore.getState().refreshAccessToken();
 assert.equal(calls,2);assert.equal(storage.get('refreshToken'),'rotated2');assert.equal(useAuthStore.getState().refreshToken,'rotated2');
 calls=0;globalThis.fetch=async()=>{calls++;return Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401})};
 await assert.rejects(useAuthStore.getState().refreshAccessToken());assert.equal(calls,1);
});
test('logout revokes stored refresh token before clearing state',async()=>{
 storage.set('refreshToken','logout-token');useAuthStore.setState({refreshToken:'logout-token',accessToken:'access',isAuthenticated:true});
 const recorded=[];globalThis.fetch=async(url,options)=>{recorded.push({url,body:JSON.parse(options.body),token:useAuthStore.getState().refreshToken});return new Response(null,{status:204})};
 await useAuthStore.getState().logout();assert.equal(recorded.length,1);assert.ok(recorded[0].url.endsWith('/api/v1/auth/logout'));assert.equal(recorded[0].body.refresh_token,'logout-token');assert.equal(recorded[0].token,'logout-token');assert.equal(useAuthStore.getState().isAuthenticated,false);assert.equal(storage.get('refreshToken'),undefined);
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
 storage.set('refreshToken','expired-access-refresh');useAuthStore.setState({refreshToken:'expired-access-refresh',accessToken:'expired',isAuthenticated:true});const recorded=[];
 globalThis.fetch=async(url,options)=>{
  recorded.push({url,body:JSON.parse(options.body)});
  if(recorded.length===1)return Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401});
  if(recorded.length===2)return Response.json({data:{access_token:'new-access',refresh_token:'current-refresh'}});
  return new Response(null,{status:204});
 };
 await useAuthStore.getState().logout();assert.equal(recorded.length,3);
 assert.ok(recorded[0].url.endsWith('/auth/logout'));assert.equal(recorded[0].body.refresh_token,'expired-access-refresh');
 assert.ok(recorded[1].url.endsWith('/auth/refresh'));assert.equal(recorded[1].body.refresh_token,'expired-access-refresh');
 assert.ok(recorded[2].url.endsWith('/auth/logout'));assert.equal(recorded[2].body.refresh_token,'current-refresh');
 assert.equal(useAuthStore.getState().isAuthenticated,false);
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
 export function useCallback(fn){return fn};export function useRef(initial){return {current:initial}};export function useEffect(){};`,'hooks');
const hooks=await import(hooksUrl);
const emptyComponentUrl=compile('export default function Component(){return null};','empty-component');
function storeModule(name, state) {
 const url=compile('export let state;export const selected=[];export function setState(value){state=value;selected.length=0};export function '+name+'(selector){const value=selector(state);selected.push(value);return value};'+name+'.getState=()=>state;',name);
 return import(url).then(module=>{if(state.selectGuild){const select=state.selectGuild;state.selectGuild=id=>{state.selectedGuildId=id;return select(id)}};module.setState(state);return {url,module}});
}
let componentSequence=0;
async function component(file, replacements, suffix='') {
 if ((file==='ServerList'||file==='AppLayout') && !replacements['@/lib/invite']) replacements['@/lib/invite']=compile(readFileSync('src/lib/invite.ts','utf8').replaceAll('"@/lib/api"',JSON.stringify(replacements['@/lib/api']??apiUrl)).replaceAll('"@/stores/guilds"',JSON.stringify(replacements['@/stores/guilds'])).replaceAll('"@/stores/channels"',JSON.stringify(replacements['@/stores/channels'])),'selection-'+(++componentSequence));
 let source=readFileSync(resolve((file==='AppLayout'?'src/layouts/':['InvitePage','LoginPage','RegisterPage','ServerAddressPage'].includes(file)?'src/pages/':'src/components/')+file+'.tsx'),'utf8')+suffix;
 source=source.replace(/from "([^"]+)"/g,(whole,name)=>{
  const target=name==='react'?(replacements.react??hooksUrl):replacements[name]??(name.startsWith('@/components/')?emptyComponentUrl:undefined);
  return target?'from '+JSON.stringify(target):whole;
 });
 // The JSX runtime import is introduced by TypeScript after import rewriting.
 const url=compile(source,file+'-component-'+(++componentSequence));const path=new URL(url);
 writeFileSync(path,readFileSync(path,'utf8').replace('"react/jsx-runtime"',JSON.stringify(jsxUrl)));
 return import(url);
}
function nodes(tree) {
 if(!tree || typeof tree!=='object')return [];
 if(Array.isArray(tree))return tree.flatMap(nodes);
 return [tree,...nodes(tree.props?.children)];
}
function element(tree, predicate) {const match=nodes(tree).find(predicate);assert.ok(match,'component element missing');return match;}
async function selectionFixture() {
 const gs={guilds:new Map([['old',{id:'old',name:'Old'}],['new',{id:'new',name:'New'}]]),selectedGuildId:'old',selectGuild(id){gs.selectedGuildId=id},fetchGuilds:async()=>{},createGuild:async()=>({id:'new'})};
 const cs={channelsByGuild:new Map([['old',[{id:'old-text',type:0}]]]),selectedChannelId:'old-text',selectChannel(id){cs.selectedChannelId=id},fetchChannels:async id=>cs.channelsByGuild.set(id,[{id:id+'-text',type:0}])};
 const g=await storeModule('useGuildsStore',gs),c=await storeModule('useChannelsStore',cs),d=await storeModule('useDMsStore',{showDMList:false,setShowDMList(){}});
 const fakeApi=compile('export class ApiClientError extends Error {} export const api={post:async()=>({id:"new"})}','selection-api-'+(++componentSequence));
 const replacements={'@/lib/api':fakeApi,'@/stores/guilds':g.url,'@/stores/channels':c.url,'@/stores/dms':d.url};
 const mod=await component('ServerList',replacements,'\nexport {CreateGuildModal,JoinGuildModal};');
 return {gs,cs,mod};
}

test('L44 create server clears the previous channel before loading',async()=>{
 const {gs,cs,mod}=await selectionFixture();let release;
 cs.fetchChannels=()=>new Promise(r=>release=r);
 hooks.reset(['New',false,'']);const pending=element(mod.CreateGuildModal({onClose(){}}),n=>n.type==='form').props.onSubmit({preventDefault(){}});
 await new Promise(setImmediate);assert.equal(gs.selectedGuildId,'new');assert.equal(cs.selectedChannelId,null);
 release();await pending;assert.equal(cs.selectedChannelId,null);
});
test('L44 join code selects the joined first text channel',async()=>{
 const {gs,cs,mod}=await selectionFixture();hooks.reset(['code',false,'']);
 await element(mod.JoinGuildModal({onClose(){}}),n=>n.type==='form').props.onSubmit({preventDefault(){}});
 assert.equal(gs.selectedGuildId,'new');assert.equal(cs.selectedChannelId,'new-text');
});
test('L44 empty server clears the previous channel',async()=>{
 const {gs,cs,mod}=await selectionFixture();cs.channelsByGuild.set('new',[{id:'voice',type:2}]);hooks.reset();
 const tree=mod.default();await element(tree,n=>n.props?.guild?.id==='new').props.onClick();
 assert.equal(gs.selectedGuildId,'new');assert.equal(cs.selectedChannelId,null);
});
test('L44 late A channels cannot replace B selection or DM view',async()=>{
 const {gs,cs,mod}=await selectionFixture();let release;
 cs.fetchChannels=id=>new Promise(r=>release=()=>{cs.channelsByGuild.set(id,[{id:'a-text',type:0}]);r()});
 cs.channelsByGuild.set('old',[{id:'b-text',type:0}]);hooks.reset();const tree=mod.default();
 element(tree,n=>n.props?.guild?.id==='new').props.onClick();
 element(tree,n=>n.props?.guild?.id==='old').props.onClick();release();await new Promise(setImmediate);
 assert.equal(gs.selectedGuildId,'old');assert.equal(cs.selectedChannelId,'b-text');
 cs.channelsByGuild.delete('new');element(tree,n=>n.props?.guild?.id==='new').props.onClick();
 element(tree,n=>n.props?.title==='Direct Messages').props.onClick();release();await new Promise(setImmediate);
 assert.equal(gs.selectedGuildId,null);assert.equal(cs.selectedChannelId,null);
});

test('invite revoke uses the versioned API path',async()=>{
 let requested;
 const fakeApi=compile('export class ApiClientError extends Error {} export const api={get:async()=>[],delete:async(path)=>{globalThis.inviteRevokePath=path}}','invite-api');
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
 const channels=await storeModule('useChannelsStore',{channelsByGuild:new Map(),selectChannel(){},fetchChannels:async id=>fetched.push(id)});
 const dms=await storeModule('useDMsStore',{setShowDMList(){}});
 const fakeApi=compile('export class ApiClientError extends Error {} export const api={post:async()=>({id:"joined-guild"})}','join-api');
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

test('message composer links uploaded ids and retains files without text',async()=>{
 const calls=[];globalThis.composerCalls=calls;
 const fakeApi=compile('export class ApiClientError extends Error {} export const api={upload:async(path,file)=>{globalThis.composerCalls.push({path,file});return {id:"9007199254740993",message_id:null}},post:async(path,body)=>{globalThis.composerCalls.push({path,body});return {id:"message"}}}','composer-api');
 const {default:MessageInput}=await component('MessageInput',{'@/lib/api':fakeApi});
 const file=new File(['hello'],'proof.txt',{type:'text/plain'});
 hooks.reset(['with attachment',false,[file]]);let tree=MessageInput({channelId:'channel'});
 element(tree,n=>n.type==='textarea').props.onKeyDown({key:'Enter',shiftKey:false,preventDefault(){}});
 await new Promise(setImmediate);
 assert.equal(calls.length,2);assert.equal(calls[0].path,'/api/v1/channels/channel/attachments');
 assert.deepEqual(calls[1].body,{content:'with attachment',attachment_ids:['9007199254740993']});assert.deepEqual(hooks.values[2],[]);
 calls.length=0;hooks.reset(['',false,[file]]);tree=MessageInput({channelId:'channel'});
 assert.equal(element(tree,n=>n.type==='p'&&n.props.children==='Add a message to send these files.').props.children,'Add a message to send these files.');
 element(tree,n=>n.type==='textarea').props.onKeyDown({key:'Enter',shiftKey:false,preventDefault(){}});
 await new Promise(setImmediate);assert.equal(calls.length,0);assert.deepEqual(hooks.values[2],[file]);delete globalThis.composerCalls;
});

test('UserSettings always navigates when disconnect or logout throws',async()=>{
 let logoutCalls=0;
 const auth=await storeModule('useAuthStore',{user:{id:'fixture-user',display_name:'Fixture'},logout:async()=>{logoutCalls++;throw new Error('logout fixture failure')}});
 const fakeApi=compile('export class ApiClientError extends Error {} export const api={};','settings-api');
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

test('READY after RESUME signals resync and retains the replacement session',async()=>{
 const {GatewayClient}=await import(compile(readFileSync(resolve('src/lib/gateway.ts'),'utf8'),'gateway'));
 const c=new GatewayClient();let resync=0;const sent=[];
 globalThis.WebSocket={OPEN:1};c.ws={readyState:1,send:raw=>sent.push(JSON.parse(raw))};c.token='test';
 c.on('RESYNC',()=>resync++);
 c.handlePayload({op:0,t:'READY',s:1,d:{session_id:'old'}});assert.equal(resync,0);
 c.handlePayload({op:0,t:'MESSAGE_CREATE',s:7,d:{}});c.handlePayload({op:10,d:{heartbeat_interval:60000}});assert.equal(sent.at(-1).op,6);assert.equal(sent.at(-1).d.seq,7);
 c.handlePayload({op:0,t:'READY',s:1,d:{session_id:'new'}});assert.equal(resync,1);assert.equal(c.sessionId,'new');assert.equal(c.sequence,1);
 c.handlePayload({op:10,d:{heartbeat_interval:60000}});assert.equal(sent.at(-1).d.session_id,'new');c.stopHeartbeat();
});

test('fallback READY refetches open channel and applies its presence snapshot',async()=>{
 const gwUrl=compile(readFileSync(resolve('src/lib/gateway.ts'),'utf8'),'dispatcher-gateway');
 const {gateway}=await import(gwUrl);const calls=[];
 globalThis.__l18Calls=calls;
 const stateUrl=compile(`const state={selectedChannelId:'50',fetchMessages:async id=>globalThis.__l18Calls.push(['messages',id]),fetchGuilds:async()=>globalThis.__l18Calls.push(['guilds']),clearPresences:()=>globalThis.__l18Calls.push(['clear']),setPresence:(id,status)=>globalThis.__l18Calls.push(['presence',id,status])};const store={getState:()=>state};export const useGuildsStore=store,useChannelsStore=store,useMembersStore=store,useRolesStore=store,useMessagesStore=store,usePresenceStore=store,useTypingStore=store,useUsersStore=store;`,'dispatcher-stores');
 const source=readFileSync(resolve('src/lib/gateway-dispatcher.ts'),'utf8').replace('"@/lib/gateway"',JSON.stringify(gwUrl)).replace(/"@\/stores\/[^"\n]+"/g,JSON.stringify(stateUrl));
 const {initGatewayDispatcher}=await import(compile(source,'dispatcher'));initGatewayDispatcher();
 gateway.resuming=true;gateway.handlePayload({op:0,t:'READY',s:1,d:{session_id:'new',guilds:[],presences:[{user_id:'200',status:'dnd'}]}});
 assert.ok(calls.some(c=>c[0]==='messages'&&c[1]==='50'));assert.ok(calls.some(c=>c[0]==='presence'&&c[1]==='200'&&c[2]==='dnd'));delete globalThis.__l18Calls;
});

test('successful resume dispatch and cleanup clear resuming',async()=>{
 const {GatewayClient}=await import(compile(readFileSync(resolve('src/lib/gateway.ts'),'utf8'),'resume-cleanup'));
 const c=new GatewayClient();let resync=0;c.on('RESYNC',()=>resync++);
 c.resuming=true;c.handlePayload({op:0,t:'MESSAGE_CREATE',s:2,d:{}});assert.equal(c.resuming,false);
 c.handlePayload({op:0,t:'READY',s:1,d:{session_id:'fresh'}});assert.equal(resync,0);
 c.resuming=true;c.cleanup();assert.equal(c.resuming,false);
 c.handlePayload({op:0,t:'READY',s:1,d:{session_id:'fresh-again'}});assert.equal(resync,0);
});

test('restore single-flights refresh, loads user, and distinguishes 401 from offline', async()=>{
 storage.set('serverUrl','http://fixture');storage.set('refreshToken','restore-token');
 useAuthStore.setState({serverUrl:'http://fixture',refreshToken:'restore-token',user:null,isAuthenticated:false});
 let refreshes=0;
 globalThis.fetch=async url=> {if(url.endsWith('/auth/refresh')) {refreshes++;await new Promise(r=>setTimeout(r,10));return Response.json({data:{access_token:'restored',refresh_token:'restore-next'}})}return Response.json({data:{id:'restored-user'}})};
 await Promise.all([useAuthStore.getState().restore(),useAuthStore.getState().restore()]);
 assert.equal(refreshes,1);assert.equal(useAuthStore.getState().isAuthenticated,true);assert.equal(useAuthStore.getState().user.id,'restored-user');
 globalThis.fetch=async()=>{throw new TypeError('offline')};await useAuthStore.getState().restore();
 assert.equal(useAuthStore.getState().restoreState,'error');assert.equal(storage.get('refreshToken'),'restore-next');
 globalThis.fetch=async()=>Response.json({error:{code:'INVALID_TOKEN',message:'expired'}},{status:401});await useAuthStore.getState().restore();
 assert.equal(useAuthStore.getState().restoreState,'ready');assertSignedOut();
});
test('logout waits for in-flight rotation and revokes its replacement',async()=>{
 setSignedInFixture();let release;const gate=new Promise(r=>release=r);const paths=[];
 globalThis.fetch=async(url,options)=>{paths.push({url,body:JSON.parse(options.body)});if(url.endsWith('/auth/refresh')){await gate;return Response.json({data:{access_token:'new',refresh_token:'replacement'}})}return new Response(null,{status:204})};
 const rotation=useAuthStore.getState().refreshAccessToken();const logout=useAuthStore.getState().logout();release();await Promise.all([rotation,logout]);
 assert.equal(paths.length,2);assert.equal(paths[1].body.refresh_token,'replacement');assertSignedOut();
});
test('password change stores tokens, wrong current password preserves session, registration sends invite',async()=>{
 setSignedInFixture();globalThis.fetch=async()=>Response.json({error:{code:'INVALID_CREDENTIALS',message:'current password is incorrect'}},{status:403});
 await assert.rejects(useAuthStore.getState().changePassword('wrong','nextpassword'));assert.equal(useAuthStore.getState().isAuthenticated,true);
 globalThis.fetch=async()=>Response.json({data:{user:{id:'fixture-user'},access_token:'changed-access',refresh_token:'changed-refresh'}});
 await useAuthStore.getState().changePassword('current','nextpassword');assert.equal(storage.get('refreshToken'),'changed-refresh');
 let body;globalThis.fetch=async(url,options)=>{body=JSON.parse(options.body);return Response.json({data:{user:{id:'new'},access_token:'a',refresh_token:'r'}})};
 await useAuthStore.getState().register('new','password','invite-code');assert.equal(body.invite_code,'invite-code');
});

// Separate module instances model tabs sharing only browser storage and Web Locks.
let tabSequence=0;
async function authTab() {
 const name='tab-'+(++tabSequence);
 const url=compile(readFileSync(resolve('src/lib/api.ts'),'utf8'),name+'-api');
 const source=readFileSync(resolve('src/stores/auth.ts'),'utf8').replace('"@/lib/api"',JSON.stringify(url)).replace('"zustand"',JSON.stringify(pathToFileURL(require.resolve('zustand')).href));
 const {useAuthStore:store}=await import(compile(source,name+'-auth'));
 return store;
}
function installLocks(locks) {
 const previous=Object.getOwnPropertyDescriptor(navigator,'locks');
 Object.defineProperty(navigator,'locks',{value:locks,configurable:true});
 return ()=>previous?Object.defineProperty(navigator,'locks',previous):delete navigator.locks;
}
function queuedLocks() {
 let tail=Promise.resolve();
 return {request(_name,action){const result=tail.then(action);tail=result.catch(()=>{});return result}};
}
function sharedSession() {storage.set('serverUrl','http://fixture');storage.set('refreshToken','shared-token')}
function pairResponse(token) {return Response.json({data:{access_token:'access',refresh_token:token}})}
function rejectedToken() {return Response.json({error:{code:'INVALID_TOKEN',message:'expired'}},{status:401})}

test('no Web Locks: losing restore cannot erase the winning tab token',async()=>{
 const reset=installLocks(undefined);
 try {
  sharedSession();const a=await authTab(),b=await authTab();let accepted=false;
  globalThis.fetch=async(url,options)=>{
   if(!url.endsWith('/auth/refresh'))return Response.json({data:{id:'shared-user'}});
   const token=JSON.parse(options.body).refresh_token;
   if(token==='shared-token' && !accepted){accepted=true;return pairResponse('winner-token')}
   if(token==='shared-token'){await new Promise(setImmediate);return rejectedToken()}
   return pairResponse('retried-token');
  };
  await Promise.all([a.getState().restore(),b.getState().restore()]);
  assert.ok(storage.get('refreshToken'),'loser erased the winner token');
  assert.equal(a.getState().isAuthenticated,true);assert.equal(b.getState().isAuthenticated,true);
 } finally {reset()}
});

test('cross-tab lock serializes reads and rotation of the shared token',async()=>{
 const reset=installLocks(queuedLocks());
 try {
  sharedSession();const a=await authTab(),b=await authTab();let expected='shared-token',calls=0;
  globalThis.fetch=async(_url,options)=>{
   const token=JSON.parse(options.body).refresh_token;await new Promise(setImmediate);
   if(token!==expected)return rejectedToken();
   expected='rotated-'+(++calls);return pairResponse(expected);
  };
  await Promise.all([a.getState().refreshAccessToken(),b.getState().refreshAccessToken()]);
  assert.equal(calls,2);assert.equal(storage.get('refreshToken'),expected);
 } finally {reset()}
});

test('refresh callers coalesce even when a lock could serialize separate calls',async()=>{
 const reset=installLocks(queuedLocks());
 try {
  sharedSession();const tab=await authTab();let calls=0;
  globalThis.fetch=async()=>{calls++;await new Promise(setImmediate);return pairResponse('coalesced')};
  await Promise.all(Array.from({length:4},()=>tab.getState().refreshAccessToken()));
  assert.equal(calls,1);
 } finally {reset()}
});

test('without Web Locks logout drains the in-flight refresh before revoking',async()=>{
 const reset=installLocks(undefined);
 try {
  sharedSession();const tab=await authTab();let release;const gate=new Promise(r=>release=r);const sent=[];
  globalThis.fetch=async(url,options)=>{
   sent.push({url,body:JSON.parse(options.body)});
   if(url.endsWith('/auth/refresh')){await gate;return pairResponse('replacement')}
   return new Response(null,{status:204});
  };
  const refresh=tab.getState().refreshAccessToken();const logout=tab.getState().logout();
  await new Promise(setImmediate);release();await Promise.all([refresh,logout]);
  assert.equal(sent[1].body.refresh_token,'replacement');assert.equal(storage.get('refreshToken'),undefined);
 } finally {reset()}
});

test('password token write waits for the shared session lock',async()=>{
 const locks=queuedLocks(),reset=installLocks(locks);
 let release;const gate=new Promise(r=>release=r);
 try {
  sharedSession();const tab=await authTab();const holding=locks.request('retrocast-session',()=>gate);
  globalThis.fetch=async()=>Response.json({data:{user:{id:'user'},access_token:'changed',refresh_token:'password-token'}});
  const change=tab.getState().changePassword('current','nextpassword');
  await new Promise(setImmediate);assert.equal(storage.get('refreshToken'),'shared-token');
  release();await holding;await change;assert.equal(storage.get('refreshToken'),'password-token');
 } finally {release();reset()}
});

test('second replacement ends in error and retains token',async()=>{
 const reset=installLocks(undefined);try{sharedSession();const a=await authTab();let k=0;
 globalThis.fetch=async()=>{storage.set('refreshToken','other'+(++k));return rejectedToken()};
 await a.getState().restore();assert.equal(a.getState().restoreState,'error');assert.equal(storage.get('refreshToken'),'other2');assert.equal(k,2);
 }finally{reset()}
});
test('request rejection tracks its attempted token despite in-memory replacement',async()=>{
 const reset=installLocks(undefined);try{sharedSession();const a=await authTab();let k=0;
 globalThis.fetch=async url=>{if(!url.endsWith('/auth/refresh'))return Response.json({data:{id:'u'}});
 if(++k===1){storage.set('refreshToken','winner');a.setState({refreshToken:'winner'});return rejectedToken()}return pairResponse('retried')};
 await a.getState().restore();assert.equal(a.getState().isAuthenticated,true);assert.equal(storage.get('refreshToken'),'retried');
 }finally{reset()}
});
test('delayed winning tab is checked before clearing on 401',async()=>{
 const reset=installLocks(undefined);try{sharedSession();const a=await authTab();let k=0;
 globalThis.fetch=async url=>{if(!url.endsWith('/auth/refresh'))return Response.json({data:{id:'u'}});
 if(++k===1){setTimeout(()=>storage.set('refreshToken','winner'),80);return rejectedToken()}return pairResponse('retried')};
 await a.getState().restore();assert.equal(a.getState().isAuthenticated,true);assert.equal(storage.get('refreshToken'),'retried');
 }finally{reset()}
});
test('late refresh cannot overwrite a password-change token without locks',async()=>{
 const reset=installLocks(undefined);try{sharedSession();const a=await authTab(),b=await authTab();let release;const gate=new Promise(r=>release=r);
 globalThis.fetch=async url=>{if(url.endsWith('/auth/refresh')){await gate;return pairResponse('revoked')}
 return Response.json({data:{user:{id:'u'},access_token:'changed',refresh_token:'password'}})};
 const rotation=b.getState().refreshAccessToken();await new Promise(setImmediate);await a.getState().changePassword('old','newpassword');release();await rotation;
 assert.equal(storage.get('refreshToken'),'password');
 }finally{reset()}
});
test('moderation popover gates actions, sends reason, and explains hierarchy',async()=>{
 const modUrl=compile('export let value={kick:true,ban:true,userId:"mod"};export function useModeration(){return value};export function setValue(v){value=v}','moderation-mock');const mod=await import(modUrl);
 const presence=await storeModule('usePresenceStore',{presences:new Map()});const roles=await storeModule('useRolesStore',{rolesByGuild:new Map()});
 const {default:Popover}=await component('UserProfilePopover',{'@/lib/api':apiUrl,'@/hooks/useModeration':modUrl,'@/stores/presence':presence.url,'@/stores/roles':roles.url});
 const prev=globalThis.window;globalThis.window={innerWidth:1200,innerHeight:900};
 const props={member:{user_id:'target',roles:[],joined_at:'2026-01-01'},displayName:'Target',username:'target',guildId:'guild',x:10,y:10,onClose(){}};
 try{
 hooks.reset();let tree=Popover(props);assert.ok(nodes(tree).some(n=>n.props?.children==='Kick'));assert.ok(nodes(tree).some(n=>n.props?.children==='Ban'));
 mod.setValue({kick:false,ban:false,userId:'plain'});hooks.reset();tree=Popover(props);assert.ok(!nodes(tree).some(n=>n.props?.children==='Ban'||n.props?.children==='Kick'));
 mod.setValue({kick:true,ban:true,userId:'mod'});hooks.reset(['ban','spam',false,'']);tree=Popover(props);
 let sent;globalThis.fetch=async(url,o)=>{sent={url,body:JSON.parse(o.body)};return new Response(null,{status:204})};
 await element(tree,n=>n.props?.children==='Confirm Ban').props.onClick();assert.ok(sent.url.endsWith('/guilds/guild/bans/target'));assert.deepEqual(sent.body,{reason:'spam'});
 globalThis.fetch=async()=>Response.json({error:{code:'ROLE_HIERARCHY',message:'too high'}},{status:403});await element(tree,n=>n.props?.children==='Confirm Ban').props.onClick();assert.match(hooks.values[3],/ranks too high/);
 hooks.reset(['kick','',false,'']);tree=Popover(props);globalThis.fetch=async(url,o)=>{sent={url,method:o.method};return new Response(null,{status:204})};await element(tree,n=>n.props?.children==='Confirm Kick').props.onClick();assert.ok(sent.url.endsWith('/members/target'));assert.equal(sent.method,'DELETE');
 }finally{globalThis.window=prev}
});
test('Bans tab renders reasons and unbans the selected user only',async()=>{
 const {default:Bans}=await component('BansTab',{'@/lib/api':apiUrl});const bans=[{user_id:'a',reason:'spam'},{user_id:'b',reason:'abuse'}];hooks.reset([bans,false,'',null,0]);const tree=Bans({guildId:'guild'});
 assert.ok(nodes(tree).some(n=>n.props?.children==='spam'));assert.ok(nodes(tree).some(n=>n.props?.children==='abuse'));
 let sent;globalThis.fetch=async(url,o)=>{sent={url,method:o.method};return new Response(null,{status:204})};await element(tree,n=>n.props?.children==='Unban').props.onClick();assert.ok(sent.url.endsWith('/bans/a'));assert.equal(sent.method,'DELETE');assert.deepEqual(hooks.values[0],[bans[1]]);
});
test('GUILD_DELETE purges selected guild channels, messages and typing, preserving other guilds',async()=>{
 const urls={};const stores={};
 for(const name of ['users','channels','messages','typing','members','roles','guilds']){
  let source=readFileSync(resolve('src/stores/'+name+'.ts'),'utf8').replace('"zustand"',JSON.stringify(pathToFileURL(require.resolve('zustand')).href)).replace('"@/lib/api"',JSON.stringify(apiUrl));
  for(const [dep,url]of Object.entries(urls))source=source.replaceAll('"@/stores/'+dep+'"',JSON.stringify(url));
  urls[name]=compile(source,'delete-'+name);stores[name]=await import(urls[name]);
 }
 const channels=stores.channels.useChannelsStore,guilds=stores.guilds.useGuildsStore,messages=stores.messages.useMessagesStore,typing=stores.typing.useTypingStore;
 const gwUrl=compile(readFileSync(resolve('src/lib/gateway.ts'),'utf8'),'delete-gateway');const {gateway}=await import(gwUrl);
 const fallback=compile('export const usePresenceStore={getState:()=>({})};','delete-presence');
 let source=readFileSync(resolve('src/lib/gateway-dispatcher.ts'),'utf8').replace('"@/lib/gateway"',JSON.stringify(gwUrl));
 for(const [name,url]of Object.entries(urls))source=source.replaceAll('"@/stores/'+name+'"',JSON.stringify(url));source=source.replace('"@/stores/presence"',JSON.stringify(fallback));
 (await import(compile(source,'delete-dispatcher'))).initGatewayDispatcher();
 channels.getState().setChannels('gone',[{id:'a',guild_id:'gone',position:0}]);channels.getState().setChannels('kept',[{id:'b',guild_id:'kept',position:0}]);
 guilds.getState().setGuild({id:'gone'});guilds.getState().setGuild({id:'kept'});guilds.getState().selectGuild('gone');channels.getState().selectChannel('a');
 messages.setState({messagesByChannel:new Map([['a',[{id:'secret'}]],['b',[{id:'other'}]]]),hasMore:new Map([['a',true],['b',true]])});typing.setState({typingByChannel:new Map([['a',new Map([['u',{}]])],['b',new Map()]])});
 gateway.handlePayload({op:0,t:'GUILD_DELETE',s:1,d:{id:'gone'}});
 assert.equal(channels.getState().selectedChannelId,null);assert.equal(guilds.getState().selectedGuildId,null);assert.equal(channels.getState().channelsByGuild.has('gone'),false);
 assert.equal(messages.getState().messagesByChannel.has('a'),false);assert.equal(messages.getState().hasMore.has('a'),false);assert.equal(typing.getState().typingByChannel.has('a'),false);
 assert.equal(channels.getState().channelsByGuild.has('kept'),true);assert.equal(messages.getState().messagesByChannel.has('b'),true);
 channels.getState().selectChannel('b');guilds.getState().selectGuild('kept');gateway.handlePayload({op:0,t:'GUILD_DELETE',s:2,d:{id:'gone'}});assert.equal(channels.getState().selectedChannelId,'b');
});
test('new roles append above manageable roles without crossing the actor boundary',async()=>{
 const {nextRolePosition}=await import(compile(readFileSync(resolve('src/lib/role-order.ts'),'utf8'),'role-order'));
 const roles=[{id:'everyone',is_default:true,position:0},{id:'a',position:1},{id:'manager',position:5},{id:'higher',position:9}];
 assert.equal(nextRolePosition(roles,true,5),10);assert.equal(nextRolePosition(roles,false,5),2);
 assert.equal(nextRolePosition([...roles,{id:'full',position:4}],false,5),null);
 assert.equal(nextRolePosition([],false,0),null);
});
test('actual moderation hook drives kick-only, ban-only, Administrator, plain, and self popovers',async()=>{
 const auth=await storeModule('useAuthStore',{user:{id:'moderator'}}),guilds=await storeModule('useGuildsStore',{guilds:new Map([['g',{owner_id:'owner'}]])}),members=await storeModule('useMembersStore',{membersByGuild:new Map([['g',new Map([['moderator',{roles:['r']} ]])]])}),roles=await storeModule('useRolesStore',{rolesByGuild:new Map()});
 let source=readFileSync(resolve('src/hooks/useModeration.ts'),'utf8').replace('"react"',JSON.stringify(hooksUrl)).replace('"@/lib/api"',JSON.stringify(apiUrl));
 for(const [name,store]of Object.entries({auth,guilds,members,roles}))source=source.replace('"@/stores/'+name+'"',JSON.stringify(store.url));
 const modUrl=compile(source,'real-moderation-hook'),presence=await storeModule('usePresenceStore',{presences:new Map()});
 const {default:Popover}=await component('UserProfilePopover',{'@/lib/api':apiUrl,'@/hooks/useModeration':modUrl,'@/stores/presence':presence.url,'@/stores/roles':roles.url});
 const previous=globalThis.window;globalThis.window={innerWidth:1200,innerHeight:900};
 try{for(const [bits,expected]of [['32',['Kick']],['64',['Ban']],['2147483648',['Kick','Ban']],['0',[]]])for(const self of [false,true]){
  roles.module.setState({rolesByGuild:new Map([['g',[{id:'r',permissions:bits,is_default:false}]]])});hooks.reset();const tree=Popover({member:{user_id:self?'moderator':'target',roles:[],joined_at:'2026-01-01'},displayName:'Target',username:'target',guildId:'g',x:1,y:1,onClose(){}});
  const buttons=nodes(tree).filter(n=>n.type==='button').map(n=>n.props.children).filter(s=>s==='Kick'||s==='Ban');assert.deepEqual(buttons,self?[]:expected,bits+' self='+self);
 }}finally{globalThis.window=previous}
});
test('copied invite route preserves code through server, register and login',async()=>{
 assert.match(readFileSync('src/App.tsx','utf8'),/path="\/invite\/:code"/);
 const auth=await storeModule('useAuthStore',{serverUrl:null,isAuthenticated:false});
 const router=compile('export const Navigate="Navigate";export const Link="Link";export const useNavigate=()=>()=>{};export const useParams=()=>({code:"copied-code"});','invite-router');
 const invite=compile('export const acceptInvite=async()=>{}','invite-action');
 const {default:Page}=await component('InvitePage',{'@/stores/auth':auth.url,'react-router':router,'@/lib/invite':invite,'@/lib/api':apiUrl});
 hooks.reset();assert.equal(Page().props.to,'/server?invite_code=copied-code');
 auth.module.setState({serverUrl:'https://fixture',isAuthenticated:false});hooks.reset();assert.equal(Page().props.to,'/register?invite_code=copied-code');
 for(const page of ['ServerAddressPage','LoginPage','RegisterPage']) assert.match(readFileSync('src/pages/'+page+'.tsx','utf8'),/encodeURIComponent\(inviteCode\)/);
});

test('invite acceptance coalesces replays, selects target and propagates invalid errors',async()=>{
 const g=await storeModule('useGuildsStore',{setGuild:x=>calls.push(['guild',x.id]),selectGuild:x=>calls.push(['select',x])});
 const c=await storeModule('useChannelsStore',{selectChannel:x=>calls.push(['channel',x]),fetchChannels:async x=>calls.push(['fetch',x]),channelsByGuild:new Map([['target',[{id:'text',type:0}]]])});
 let calls=[];
 const source=readFileSync('src/lib/invite.ts','utf8').replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"@/stores/guilds"',JSON.stringify(g.url)).replace('"@/stores/channels"',JSON.stringify(c.url));
 const {acceptInvite}=await import(compile(source,'real-invite'));
 let requests=0;globalThis.fetch=async()=>{requests++;await new Promise(setImmediate);return Response.json({data:{id:'target'}})};
 await Promise.all([acceptInvite('copied'),acceptInvite('copied')]);assert.equal(requests,1);
 assert.deepEqual(calls,[['guild','target'],['select','target'],['channel',null],['channel','text']]);
 calls=[];globalThis.fetch=async()=>Response.json({error:{code:'EXPIRED',message:'invite has expired'}},{status:410});
 await assert.rejects(acceptInvite('expired'),/expired/);assert.deepEqual(calls,[]);
});

test('already-member invite previews once, loads missing guild and selects its channel',async()=>{
 let selected=[],calls=[];
 const g=await storeModule('useGuildsStore',{guilds:new Map(),setGuild:x=>selected.push(['guild',x.id]),selectGuild:x=>selected.push(['select',x])});
 const c=await storeModule('useChannelsStore',{selectChannel:x=>selected.push(['channel',x]),fetchChannels:async()=>{},channelsByGuild:new Map([['invited',[{id:'first-text',type:0}]]])});
 const source=readFileSync('src/lib/invite.ts','utf8').replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"@/stores/guilds"',JSON.stringify(g.url)).replace('"@/stores/channels"',JSON.stringify(c.url));
 const {acceptInvite}=await import(compile(source,'member-invite'));
 globalThis.fetch=async(url,o)=>{calls.push([o.method,url]);if(o.method==='POST')return Response.json({error:{code:'ALREADY_MEMBER',message:'already a member'}},{status:409});return Response.json({data:url.endsWith('/invites/copied')?{guild_id:'invited'}:{id:'invited'}})};
 await Promise.all([acceptInvite('copied'),acceptInvite('copied')]);
 assert.deepEqual(calls.map(([m,u])=>[m,new URL(u,'http://fixture').pathname]),[['POST','/api/v1/invites/copied'],['GET','/api/v1/invites/copied'],['GET','/api/v1/guilds/invited']]);
 assert.deepEqual(selected,[['guild','invited'],['select','invited'],['channel',null],['channel','first-text']]);
 calls=[];selected=[];globalThis.fetch=async(url,o)=>{calls.push(o.method);return Response.json({error:{code:'BANNED',message:'you are banned'}},{status:403})};
 await assert.rejects(acceptInvite('banned'),/banned/);assert.deepEqual(calls,['POST']);assert.deepEqual(selected,[]);
});

test('login submit preserves invite destination and registration never reaccepts joined membership',async()=>{
 let destinations=[],logins=0,registers=0,selected=[];
 const router=compile('export const Link="Link";export const useNavigate=()=>globalThis.l37Navigate;export const useSearchParams=()=>[new URLSearchParams("invite_code=copied-code")];','post-auth-router');
 globalThis.l37Navigate=path=>destinations.push(path);
 const a=await storeModule('useAuthStore',{login:async()=>{logins++},register:async(_u,_p,code)=>{assert.equal(code,'copied-code');registers++}});
 const g=await storeModule('useGuildsStore',{fetchGuilds:async()=>{},guilds:new Map([['invited',{id:'invited'}]])});
 globalThis.l37Select=async guild=>selected.push(guild.id);
 const invite=compile('export const selectInviteGuild=g=>globalThis.l37Select(g);','post-register-select');
 const {default:Login}=await component('LoginPage',{'@/stores/auth':a.url,'react-router':router,'@/lib/api':apiUrl});
 hooks.reset(['user','password','','false']);await Login().props.onSubmit({preventDefault(){}});assert.equal(logins,1);assert.deepEqual(destinations,['/invite/copied-code']);
 const {default:Register}=await component('RegisterPage',{'@/stores/auth':a.url,'react-router':router,'@/lib/api':apiUrl,'@/stores/guilds':g.url,'@/lib/invite':invite});
 destinations=[];hooks.reset(['copied-code','new-user','password','',false]);await Register().props.onSubmit({preventDefault(){}});
 assert.equal(registers,1);assert.deepEqual(selected,['invited']);assert.deepEqual(destinations,['/app']);
 g.module.setState({fetchGuilds:async()=>{},guilds:new Map()});destinations=[];hooks.reset(['copied-code','open-user','password','',false]);await Register().props.onSubmit({preventDefault(){}});assert.deepEqual(destinations,['/invite/copied-code']);
 delete globalThis.l37Navigate;delete globalThis.l37Select;
});

test('successful invite history state restores its guild after a redirect-time reload',async()=>{
 const effectUrl=compile('export const effects=[];export function useEffect(fn){effects.push(fn)};export const useState=v=>[v,()=>{}];export const useSyncExternalStore=()=>true;','invite-effects');const effect=await import(effectUrl);
 const a=await storeModule('useAuthStore',{isAuthenticated:true,accessToken:'fixture',serverUrl:'https://fixture'});
 let navigation,selectedGuild,selectedChannel;
 globalThis.l37Redirect=(...args)=>navigation=args;
 const router=compile('export const Link="Link",Navigate="Navigate",Outlet="Outlet";export const useNavigate=()=>globalThis.l37Redirect;export const useParams=()=>({code:"copied"});export const useLocation=()=>({state:globalThis.l37History});','invite-history-router');
 const invite=compile('export const ApiClientError=Error;export const acceptInvite=async()=>({id:"target"});','invite-history-accept');
 const {default:Invite}=await component('InvitePage',{react:effectUrl,'@/stores/auth':a.url,'react-router':router,'@/lib/invite':invite,'@/lib/api':apiUrl});
 Invite();effect.effects.shift()();await new Promise(setImmediate);assert.deepEqual(navigation,['/app',{replace:true,state:{inviteGuildId:'target'}}]);
 globalThis.l37History=navigation[1].state;effect.effects.length=0;
 const g=await storeModule('useGuildsStore',{guilds:new Map([['first',{id:'first'}],['target',{id:'target'}]]),selectedGuildId:null,fetchGuilds:async()=>{},selectGuild:id=>selectedGuild=id});
 const c=await storeModule('useChannelsStore',{fetchChannels:async()=>{},channelsByGuild:new Map([['target',[{id:'target-text',type:0}]]]),selectChannel:id=>selectedChannel=id});
 const d=await storeModule('useDMsStore',{fetchDMs:async()=>{}});
 const gateway=compile('export const gateway={connect(){},disconnect(){}}','reload-gateway');const dispatcher=compile('export const initGatewayDispatcher=()=>{}','reload-dispatcher');
 const {default:Layout}=await component('AppLayout',{react:effectUrl,'react-router':router,'@/stores/auth':a.url,'@/stores/guilds':g.url,'@/stores/channels':c.url,'@/stores/dms':d.url,'@/lib/gateway':gateway,'@/lib/gateway-dispatcher':dispatcher});
 Layout();effect.effects[0]();await new Promise(setImmediate);assert.equal(selectedGuild,'target');assert.equal(selectedChannel,'target-text');
 delete globalThis.l37History;delete globalThis.l37Redirect;
});


test('Pilot already-member store hit selects the previewed guild',async()=>{
 const wrong={id:'wrong'},target={id:'target'},selected=[];
 const g=await storeModule('useGuildsStore',{guilds:new Map([['wrong',wrong],['target',target]]),setGuild:x=>assert.equal(x,target),selectGuild:id=>selected.push(id)});
 const c=await storeModule('useChannelsStore',{selectChannel(){},fetchChannels:async id=>assert.equal(id,'target'),channelsByGuild:new Map()});
 const source=readFileSync('src/lib/invite.ts','utf8').replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"@/stores/guilds"',JSON.stringify(g.url)).replace('"@/stores/channels"',JSON.stringify(c.url));
 const {acceptInvite}=await import(compile(source,'pilot-store-hit'));const calls=[];
 globalThis.fetch=async(url,o)=>{calls.push(o.method);assert.ok(url.endsWith('/invites/code'));return o.method==='POST'?Response.json({error:{code:'ALREADY_MEMBER',message:'member'}},{status:409}):Response.json({data:{guild_id:'target'}})};
 assert.equal(await acceptInvite('code'),target);assert.deepEqual(selected,['target']);assert.deepEqual(calls,['POST','GET']);
});
test('Pilot server address persists and submit opens login',async()=>{
 let destination;globalThis.l40Navigate=p=>destination=p;
 const router=compile('export const useNavigate=()=>globalThis.l40Navigate;export const useSearchParams=()=>[new URLSearchParams()];','pilot-server-router');
 const a=await storeModule('useAuthStore',useAuthStore.getState());
 const {default:Page}=await component('ServerAddressPage',{'@/stores/auth':a.url,'react-router':router});
 hooks.reset(['https://chat.example']);Page().props.onSubmit({preventDefault(){}});
 assert.equal(storage.get('serverUrl'),'https://chat.example');assert.equal(destination,'/login');delete globalThis.l40Navigate;
});
test('Pilot expired channel refreshes once and replaces stored token',async()=>{
 storage.set('refreshToken','old');useAuthStore.setState({refreshToken:'old'});let refreshes=0,requests=0;
 globalThis.fetch=async url=>{if(url.endsWith('/auth/refresh')){refreshes++;return Response.json({data:{access_token:'fresh',refresh_token:'rotated'}})}requests++;return requests===1?Response.json({error:{code:'UNAUTHORIZED',message:'expired'}},{status:401}):Response.json({data:{id:'channel'}})};
 assert.equal((await api.get('/api/v1/channels/channel')).id,'channel');assert.equal(refreshes,1);assert.equal(requests,2);assert.equal(storage.get('refreshToken'),'rotated');
});
test('Pilot composer sends hi, clears and throttles twenty seconds of input',async()=>{
 const calls=[];globalThis.l40Posts=calls;const a=compile('export const api={post:async(...args)=>{globalThis.l40Posts.push(args);return {}}};','pilot-composer-api');
 const {default:Composer}=await component('MessageInput',{'@/lib/api':a});hooks.reset(['hi',false,[]]);let tree=Composer({channelId:'channel'});
 element(tree,n=>n.type==='textarea').props.onKeyDown({key:'Enter',shiftKey:false,preventDefault(){}});await new Promise(setImmediate);
 assert.deepEqual(calls,[['/api/v1/channels/channel/messages',{content:'hi',attachment_ids:[]}]]);assert.equal(hooks.values[0],'');
 const now=Date.now;try{calls.length=0;hooks.reset();tree=Composer({channelId:'channel'});const input=element(tree,n=>n.type==='textarea');for(let i=0;i<20;i++){Date.now=()=>100000+i*1000;input.props.onChange({target:{value:'hi'+i}})}assert.equal(calls.length,3);assert.ok(calls.every(p=>p[0]==='/api/v1/channels/channel/typing'));}finally{Date.now=now;delete globalThis.l40Posts}
});
test('Pilot typing expires at eight seconds and renewed signals extend it',async()=>{
 const source=readFileSync('src/stores/typing.ts','utf8').replace('"zustand"',JSON.stringify(pathToFileURL(require.resolve('zustand')).href));const {useTypingStore:s}=await import(compile(source,'pilot-typing'));
 const now=Date.now,timeout=globalThis.setTimeout;let clock=0,timers=[];
 try{Date.now=()=>clock;globalThis.setTimeout=(fn,ms)=>{assert.equal(ms,8000);timers.push(fn)};
 s.getState().setTyping('c','u');clock=7900;assert.ok(s.getState().typingByChannel.get('c').has('u'));clock=8000;timers.shift()();assert.equal(s.getState().typingByChannel.get('c').has('u'),false);
 clock=10000;s.getState().setTyping('c','u');clock=17000;s.getState().setTyping('c','u');clock=18000;timers.shift()();assert.ok(s.getState().typingByChannel.get('c').has('u'));clock=25000;timers.shift()();assert.equal(s.getState().typingByChannel.get('c').has('u'),false);
 }finally{Date.now=now;globalThis.setTimeout=timeout}
});
test('Pilot member names resolve nickname then display name then username',async()=>{
 const member={guild_id:'g',user_id:'200',roles:[],user:{id:'200',username:'bob',display_name:null}};
 const m=await storeModule('useMembersStore',{membersByGuild:new Map([['g',new Map([['200',member]])]]),fetchMembers:async()=>{}}),r=await storeModule('useRolesStore',{rolesByGuild:new Map(),fetchRoles:async()=>{}}),p=await storeModule('usePresenceStore',{presences:new Map()}),u=await storeModule('useUsersStore',{users:new Map()}),typing=await storeModule('useTypingStore',{typingByChannel:new Map([['c',new Map([['200',{}]])]])}),a=await storeModule('useAuthStore',{user:{id:'100'}}),g=await storeModule('useGuildsStore',{selectedGuildId:'g'});
 const replacements={'@/stores/members':m.url,'@/stores/roles':r.url,'@/stores/presence':p.url,'@/stores/users':u.url,'@/stores/typing':typing.url,'@/stores/auth':a.url,'@/stores/guilds':g.url};
 const {default:Indicator}=await component('TypingIndicator',replacements),{default:List}=await component('MemberList',replacements);
 for(const expected of ['bob','Bobby','Nick']){if(expected==='Bobby')member.user.display_name='Bobby';if(expected==='Nick')member.nickname='Nick';hooks.reset();assert.ok(nodes(Indicator({channelId:'c'})).some(n=>Array.isArray(n.props?.children)&&n.props.children.includes(expected+' is typing...')));hooks.reset();assert.ok(nodes(List({guildId:'g'})).some(n=>n.props?.info?.displayName===expected));}
});
test('Pilot member responses and events hydrate the shared users cache',async()=>{
 const zustand=JSON.stringify(pathToFileURL(require.resolve('zustand')).href),usersUrl=compile(readFileSync('src/stores/users.ts','utf8').replace('"zustand"',zustand),'pilot-users');const {useUsersStore:u}=await import(usersUrl);
 const source=readFileSync('src/stores/members.ts','utf8').replace('"zustand"',zustand).replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"@/stores/users"',JSON.stringify(usersUrl));const {useMembersStore:m}=await import(compile(source,'pilot-members'));
 const member={guild_id:'g',user_id:'200',roles:[],user:{id:'200',username:'bob',display_name:null}};globalThis.fetch=async()=>Response.json({data:[member]});await m.getState().fetchMembers('g');assert.equal(u.getState().users.get('200').username,'bob');m.getState().setMember({...member,user:{...member.user,display_name:'Bobby'}});assert.equal(u.getState().users.get('200').display_name,'Bobby');let requests=0;globalThis.fetch=async url=>{requests++;assert.ok(url.endsWith('/guilds/g/members/300'));await new Promise(setImmediate);return Response.json({data:{...member,user_id:'300',user:{...member.user,id:'300',username:'carol'}}})};await Promise.all([m.getState().fetchMember('g','300'),m.getState().fetchMember('g','300')]);assert.equal(requests,1);assert.equal(u.getState().users.get('300').username,'carol');
});


test('Pilot stored server skips the address page on return',async()=>{
 const a=await storeModule('useAuthStore',{restoreState:'ready',restore(){},isAuthenticated:false,serverUrl:storage.get('serverUrl')});
 const router=compile('export const BrowserRouter="BrowserRouter",Routes="Routes",Route="Route",Navigate="Navigate";','pilot-app-router');
 let source=readFileSync('src/App.tsx','utf8').replace('"react"',JSON.stringify(hooksUrl)).replace('"react-router"',JSON.stringify(router)).replace('"@/stores/auth"',JSON.stringify(a.url)).replace(/"@\/(?:pages|layouts|components)\/[^"\n]+"/g,JSON.stringify(emptyComponentUrl));
 let url=compile(source,'pilot-app');writeFileSync(new URL(url),readFileSync(new URL(url),'utf8').replace('"react/jsx-runtime"',JSON.stringify(jsxUrl)));
 const {default:App}=await import(url);hooks.reset();const route=element(App(),n=>n.props?.path==='*');assert.equal(route.props.element.props.to,'/login');
});
test('Pilot registration explains invite-required and closed errors',async()=>{
 const router=compile('export const Link="Link";export const useNavigate=()=>()=>{};export const useSearchParams=()=>[new URLSearchParams()];','pilot-register-router');
 const g=await storeModule('useGuildsStore',{}),invite=compile('export const selectInviteGuild=()=>{}','pilot-register-invite');
 for(const [code,message]of [['INVITE_REQUIRED','An invite is required to register.'],['REGISTRATION_CLOSED','Registration is closed.']]){
 const a=await storeModule('useAuthStore',{register:async()=>{throw new ApiClientError(code,'fixture',403)}});
 const {default:Page}=await component('RegisterPage',{'react-router':router,'@/stores/auth':a.url,'@/stores/guilds':g.url,'@/lib/invite':invite,'@/lib/api':apiUrl});hooks.reset(['','bob','password','',false]);await Page().props.onSubmit({preventDefault(){}});assert.equal(hooks.values[3],message);
 }
});
test('Pilot markdown displays hostile markup literally without active attributes',async()=>{
 let source=readFileSync('src/components/MarkdownContent.tsx','utf8').replace(/from "(react-markdown|remark-gfm|react)"/g,(_,name)=>'from '+JSON.stringify(pathToFileURL(require.resolve(name)).href));
 const url=compile(source,'pilot-markdown');writeFileSync(new URL(url),readFileSync(new URL(url),'utf8').replace('"react/jsx-runtime"',JSON.stringify(pathToFileURL(require.resolve('react/jsx-runtime')).href)));
 const {default:Markdown}=await import(url),{renderToStaticMarkup}=require('react-dom/server'),{createElement}=require('react');
 const html=renderToStaticMarkup(createElement(Markdown,{content:'<img src=x onerror=alert(1)>'}));assert.ok(html.includes('&lt;img src=x onerror=alert(1)&gt;'));assert.doesNotMatch(html,/<img\b|<[^>]+\son\w+=/i);
});
test('Pilot leave removes the selected non-owner guild',async()=>{
 const deps=compile('const s={channelsByGuild:new Map(),messagesByChannel:new Map(),hasMore:new Map(),typingByChannel:new Map()};const store={getState:()=>s,setState:fn=>fn(s)};export const useChannelsStore=store,useMessagesStore=store,useTypingStore=store;','pilot-leave-deps');
 const source=readFileSync('src/stores/guilds.ts','utf8').replace(/"@\/stores\/[^"\n]+"/g,JSON.stringify(deps)).replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"zustand"',JSON.stringify(pathToFileURL(require.resolve('zustand')).href));
 const guildUrl=compile(source,'pilot-leave-store'),{useGuildsStore:g}=await import(guildUrl),guild={id:'g',name:'Guild',owner_id:'owner'};g.getState().setGuild(guild);g.getState().selectGuild('g');
 const a=await storeModule('useAuthStore',{user:{id:'other'}}),moderation=compile('export const useModeration=()=>({canBan:false,canManageRoles:false})','pilot-leave-moderation');
 const guildSnapshot=await storeModule('useGuildsStore',g.getState());const {default:Modal}=await component('GuildSettingsModal',{'@/stores/auth':a.url,'@/stores/guilds':guildSnapshot.url,'@/hooks/useModeration':moderation,'@/lib/api':apiUrl});
 let path;globalThis.fetch=async url=>{path=url;return new Response(null,{status:204})};hooks.reset();const tree=Modal({guild,onClose(){}});await element(tree,n=>n.type==='button'&&n.props.children==='Leave Server').props.onClick();assert.ok(path.endsWith('/api/v1/guilds/g/members/@me'));assert.equal(g.getState().guilds.has('g'),false);
});
test('Pilot gateway channel, membership and idle updates drive real stores',async()=>{
 const zustand=JSON.stringify(pathToFileURL(require.resolve('zustand')).href),urls={},mods={};
 for(const name of ['users','members','channels','presence']){let source=readFileSync('src/stores/'+name+'.ts','utf8').replace('"zustand"',zustand).replace('"@/lib/api"',JSON.stringify(apiUrl));if(urls.users)source=source.replace('"@/stores/users"',JSON.stringify(urls.users));urls[name]=compile(source,'pilot-event-'+name);mods[name]=await import(urls[name]);}
 const gw=compile('const callbacks=new Map();export const gateway={on:(name,fn)=>callbacks.set(name,fn),emit:(name,data)=>callbacks.get(name)(data)};','pilot-event-gw');const {gateway}=await import(gw);
 const noop=compile('export const useGuildsStore={},useRolesStore={},useMessagesStore={},useTypingStore={};','pilot-unused-stores');
 const source=readFileSync('src/lib/gateway-dispatcher.ts','utf8').replace('"@/lib/gateway"',JSON.stringify(gw)).replace(/"@\/stores\/([^"\n]+)"/g,(_,name)=>JSON.stringify(urls[name]??noop));
 (await import(compile(source,'pilot-event-dispatcher'))).initGatewayDispatcher();
 const ch={id:'c',guild_id:'g',name:'random',type:0};gateway.emit('CHANNEL_CREATE',ch);assert.equal(mods.channels.useChannelsStore.getState().channelsByGuild.get('g')[0].name,'random');gateway.emit('CHANNEL_DELETE',{id:'c',guild_id:'g'});assert.deepEqual(mods.channels.useChannelsStore.getState().channelsByGuild.get('g'),[]);
 const member={guild_id:'g',user_id:'bob',roles:[],user:{id:'bob',username:'bob',display_name:null}};gateway.emit('GUILD_MEMBER_ADD',member);assert.equal(mods.users.useUsersStore.getState().users.get('bob').username,'bob');gateway.emit('GUILD_MEMBER_REMOVE',{guild_id:'g',user_id:'bob'});assert.equal(mods.members.useMembersStore.getState().membersByGuild.get('g').has('bob'),false);
 gateway.emit('PRESENCE_UPDATE',{user_id:'bob',status:'idle'});const snapshot=await storeModule('usePresenceStore',mods.presence.usePresenceStore.getState());const {default:Dot}=await component('PresenceDot',{'@/stores/presence':snapshot.url});assert.ok(Dot({userId:'bob'}).props.className.includes('bg-[#f0b232]'));
});


test('Pilot channel deletion accepts 204 and closes without an error',async()=>{
 const c=await storeModule('useChannelsStore',{removeChannel:(id,guild)=>assert.deepEqual([id,guild],['c','g'])});let closed=false,path;
 const {default:Modal}=await component('ChannelModal',{'@/stores/channels':c.url,'@/lib/api':apiUrl});globalThis.fetch=async url=>{path=url;return new Response(null,{status:204})};const confirm=globalThis.confirm;
 try{globalThis.confirm=()=>true;hooks.reset();const tree=Modal({guildId:'g',channel:{id:'c'},onClose(){closed=true}});await element(tree,n=>n.type==='button'&&n.props.children==='Delete Channel').props.onClick();assert.ok(path.endsWith('/api/v1/channels/c'));assert.equal(closed,true);assert.equal(hooks.values[4],'');}finally{globalThis.confirm=confirm}
});


test('Pilot uncached typing member fetches the public member endpoint',async()=>{
 const effects=compile('export let effect;export const useEffect=fn=>effect=fn;','pilot-typing-effect');const calls=[];
 const m=await storeModule('useMembersStore',{membersByGuild:new Map(),fetchMember:async(...args)=>calls.push(args)}),u=await storeModule('useUsersStore',{users:new Map()}),t=await storeModule('useTypingStore',{typingByChannel:new Map([['c',new Map([['300',{}]])]])}),a=await storeModule('useAuthStore',{user:{id:'100'}}),g=await storeModule('useGuildsStore',{selectedGuildId:'g'});
 const {default:Indicator}=await component('TypingIndicator',{react:effects,'@/stores/members':m.url,'@/stores/users':u.url,'@/stores/typing':t.url,'@/stores/auth':a.url,'@/stores/guilds':g.url});Indicator({channelId:'c'});(await import(effects)).effect();assert.deepEqual(calls,[['g','300']]);
 u.module.setState({users:new Map([['300',{username:'carol',display_name:null}]])});calls.length=0;const tree=Indicator({channelId:'c'});(await import(effects)).effect();assert.deepEqual(calls,[]);assert.ok(nodes(tree).some(n=>Array.isArray(n.props?.children)&&n.props.children.includes('carol is typing...')));
});


test('Typing member lookup caches 404 and 403 across twenty unrelated updates',async()=>{
 for(const status of [404,403]) await checkTypingLookup(status);
});
test('Typing member without public user is fetched once across store changes',async()=>{
 await checkTypingLookup(200);
});
async function checkTypingLookup(status) {
 const zustand=JSON.stringify(pathToFileURL(require.resolve('zustand')).href);
 const usersUrl=compile(readFileSync('src/stores/users.ts','utf8').replace('"zustand"',zustand),'r2-users-'+status);
 const {useUsersStore:u}=await import(usersUrl);
 const memberSource=readFileSync('src/stores/members.ts','utf8').replace('"zustand"',zustand).replace('"@/lib/api"',JSON.stringify(apiUrl)).replace('"@/stores/users"',JSON.stringify(usersUrl));
 const {useMembersStore:m}=await import(compile(memberSource,'r2-members-'+status));
 const hooksUrl2=compile('export let effect,deps;export const useEffect=(f,d)=>{effect=f;deps=d};','r2-effect-'+status),effect=await import(hooksUrl2);
 globalThis.l40R2LookupStores={members:m,users:u};
 const wrap=(name,store)=>compile('export const '+name+'=sel=>sel(globalThis.l40R2LookupStores.'+store+'.getState());','r2-wrap-'+store+'-'+status);
 const typing=await storeModule('useTypingStore',{typingByChannel:new Map([['c',new Map([['300',{timestamp:Date.now()}]])]])});
 const a=await storeModule('useAuthStore',{user:{id:'100'}}),g=await storeModule('useGuildsStore',{selectedGuildId:'g'});
 const {default:Indicator}=await component('TypingIndicator',{react:hooksUrl2,'@/stores/members':wrap('useMembersStore','members'),'@/stores/users':wrap('useUsersStore','users'),'@/stores/typing':typing.url,'@/stores/auth':a.url,'@/stores/guilds':g.url});
 let calls=0,active=true,previous;
 globalThis.fetch=async url=>{calls++;assert.ok(url.endsWith('/guilds/g/members/300'));await new Promise(setImmediate);return status===200?Response.json({data:{guild_id:'g',user_id:'300',roles:[]}}):Response.json({error:{code:'NOT_FOUND',message:'gone'}},{status})};
 const commit=()=>{if(!active)return;Indicator({channelId:'c'});const changed=!previous||effect.deps.some((x,i)=>!Object.is(x,previous[i]));previous=effect.deps;if(changed)effect.effect()};
 const schedule=()=>setImmediate(commit),unsub=[m.subscribe(schedule),u.subscribe(schedule)];
 const flush=async()=>{for(let i=0;i<5;i++)await new Promise(setImmediate)};
 try {
 commit();await flush();
 for(let i=0;i<20;i++){u.getState().cacheUser('other'+i,{username:'other',display_name:null,avatar_hash:null});await flush()}
 assert.equal(calls,1,'lookup must not repeat for '+status);
 if(status!==200){const now=Date.now;try{Date.now=()=>now()+8001;await m.getState().fetchMember('g','300').catch(()=>{});assert.equal(calls,2,'negative cache must allow a later typing entry to retry')}finally{Date.now=now}}
 } finally {active=false;unsub.forEach(fn=>fn());await flush();delete globalThis.l40R2LookupStores}
}

} catch (error) {
 cleanup();
 throw error;
}
