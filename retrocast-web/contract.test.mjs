import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import assert from 'node:assert/strict';
import test, { after } from 'node:test';
import ts from 'typescript';
const require = createRequire(import.meta.url);
const dir = mkdtempSync(join(tmpdir(), 'l13-web-test-'));
after(() => rmSync(dir, {recursive:true,force:true}));
const storage = new Map();
globalThis.localStorage = {getItem:k=>storage.get(k)??null,setItem:(k,v)=>storage.set(k,v),removeItem:k=>storage.delete(k)};
function compile(source, name) {
 const file=join(dir,name+'.mjs');
 writeFileSync(file, ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.ES2022}}).outputText);
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
