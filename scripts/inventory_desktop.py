#!/usr/bin/env python3
"""Inventory shipped native command declarations without reading user profiles."""
import argparse, json, re, struct
from pathlib import Path

def inventory(root):
    found={}
    pattern=re.compile(rb'([0-9]{1,8})\|([A-Za-z0-9_.]+(?:Request|Req))\|([A-Za-z0-9_.]+(?:Response|Resp))(?:\|([0-9]+)\|([A-Z][A-Z0-9_]+))?')
    for archive in sorted(root.glob('*.asar')):
        b=archive.read_bytes()
        if len(b)<16:continue
        base=8+struct.unpack_from('<I',b,4)[0]
        header=json.loads(b[16:16+struct.unpack_from('<I',b,12)[0]])
        def walk(node,path=''):
            for name,entry in node.get('files',{}).items():
                current=f'{path}/{name}'.lstrip('/')
                if 'files' in entry:yield from walk(entry,current)
                elif name.endswith('.js') and 'offset' in entry and not entry.get('unpacked'):
                    start=base+int(entry['offset']);yield current,b[start:start+entry['size']]
        for path,content in walk(header):
            for match in pattern.finditer(content):
                wire=match.group().decode()
                item=found.setdefault(wire,{'command':wire,'name':match[5].decode() if match[5] else match[2].decode().rsplit('.',1)[-1],'request':match[2].decode(),'response':match[3].decode(),'sources':[],'status':'declaration-only'})
                source={'archive':archive.name,'file':path,'offset':match.start()}
                if source not in item['sources']:item['sources'].append(source)
    return sorted(found.values(),key=lambda r:r['name'])
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--root',type=Path,default=Path('/Applications/LarkSuite.app/Contents/Frameworks/Lark Framework.framework/Versions/Current/Resources/webcontent'));p.add_argument('--output',required=True,type=Path);a=p.parse_args()
    result=inventory(a.root);a.output.parent.mkdir(parents=True,exist_ok=True);a.output.write_text(json.dumps({'note':'Static declarations are not proof of availability, permission, request schema, or successful execution. Not exhaustive of native C++ or dynamically loaded features.','operations':result},indent=2)+'\n');print(f'{len(result)} unique native command declarations written to {a.output}')
