const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const root=path.join(__dirname,'../internal/desktop/modules');
const declarations=new Set(JSON.parse(fs.readFileSync(path.join(__dirname,'../internal/desktop/catalog.json'))).operations.map(x=>x.command));
test('curated read registrations match manifests and shipped declarations',()=>{
 const all=new Set();
 for(const dir of fs.readdirSync(root)){
  const base=path.join(root,dir);if(!fs.statSync(base).isDirectory())continue;
  const specs=JSON.parse(fs.readFileSync(path.join(base,'manifest.json')));
  const {operations}=require(path.join(base,'module.js'));
  assert.deepEqual(Object.keys(operations).sort(),specs.map(x=>x.name).sort(),dir);
  for(const spec of specs){
   assert(!all.has(spec.name),'duplicate '+spec.name);all.add(spec.name);
   assert.equal(spec.readOnly,true);assert.equal(spec.verification,'schema-verified');assert(spec.evidence.length>0);
   assert.equal(operations[spec.name].transport || 'sdk',spec.transport || 'sdk');
   assert(declarations.has(operations[spec.name].command) || (spec.readOnly && spec.name==='workspacenext.navigation-layout' && spec.transport==='shell-navigation' && operations[spec.name].command==='ShellAPI.app.navigation.getNavigationInfo'),'unknown wire declaration '+spec.name);
   assert.equal(typeof operations[spec.name].request,'function');assert.equal(typeof operations[spec.name].project,'function');
  }
 }
});

test('curated mutation registrations match declarations and effects',()=>{
 const all=new Set();
 for(const dir of fs.readdirSync(root)){
  const base=path.join(root,dir);if(!fs.statSync(base).isDirectory()||!fs.existsSync(path.join(base,'mutation-manifest.json')))continue;
  const specs=JSON.parse(fs.readFileSync(path.join(base,'mutation-manifest.json')));
  const {operations}=require(path.join(base,'mutations.js'));
  assert.deepEqual(Object.keys(operations).sort(),specs.map(x=>x.name).sort(),dir);
  for(const spec of specs){
   assert(!all.has(spec.name));all.add(spec.name);
   assert.equal(spec.readOnly,false);assert(['draft','settings','outbound'].includes(spec.effect));
   assert.equal(spec.verification,'schema-verified');assert(spec.evidence.length>0);
   assert.equal(operations[spec.name].transport || 'sdk',spec.transport || 'sdk');
   assert(declarations.has(operations[spec.name].command) || (spec.readOnly && spec.name==='workspacenext.navigation-layout' && spec.transport==='shell-navigation' && operations[spec.name].command==='ShellAPI.app.navigation.getNavigationInfo'),'unknown wire declaration '+spec.name);
   assert.equal(typeof operations[spec.name].request,'function');assert.equal(typeof operations[spec.name].project,'function');
  }
 }
});
