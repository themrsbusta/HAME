"""Package HAME.ico as a native COFF .rsrc object using installed clang.
No network/dependency install required. Re-run only when the icon changes.
"""
from pathlib import Path
import struct,subprocess,shutil
p=Path(__file__).resolve().parents[2]
raw=(p/'assets'/'HAME.ico').read_bytes();reserved,kind,count=struct.unpack_from('<HHH',raw);assert kind==1
images=[];group=bytearray(struct.pack('<HHH',0,1,count))
for i in range(count):
 w,h,colors,res,planes,bits,size,offset=struct.unpack_from('<BBBBHHII',raw,6+i*16)
 images.append(raw[offset:offset+size]);group+=struct.pack('<BBBBHHIH',w,h,colors,res,planes,bits,size,i+1)
def utf16(text):return (text+'\0').encode('utf-16le')
def block(key,value=b'',children=b'',text=False):
 header=struct.pack('<HHH',0,len(value)//2 if text else len(value),1 if text else 0)+utf16(key)
 header+=b'\0'*((-len(header))%4);data=header+value;data+=b'\0'*((-len(data))%4);data+=children
 return struct.pack('<H',len(data))+data[2:]
strings=b''
for key,value in [('CompanyName','Busta Supastar'),('FileDescription','HAME 1.1 beta'),('FileVersion','1.1.0.1'),('ProductName','HAME'),('ProductVersion','1.1-beta.1'),('LegalCopyright','Copyright (c) 2026 Busta Supastar; BSD-3-Clause')]:
 child=block(key,utf16(value),text=True);strings+=child+b'\0'*((-len(child))%4)
fixed=struct.pack('<13I',0xFEEF04BD,0x10000,0x10001,1,0x10001,1,0x3f,0,0x40004,1,0,0,0)
version=block('VS_VERSION_INFO',fixed,block('StringFileInfo',children=block('040904B0',children=strings))+block('VarFileInfo',children=block('Translation',struct.pack('<HH',0x409,1200))))
payloads=images+[bytes(group),version]
tree={3:{i+1:{1033:i} for i in range(count)},14:{1:{1033:count}},16:{1:{1033:count+1}}}
dirs=[];offsets={};cursor=0
def allocate(node):
 global cursor
 offsets[id(node)]=cursor;dirs.append(node);cursor+=16+8*len(node)
 for _,v in sorted(node.items()):
  if isinstance(v,dict):allocate(v)
allocate(tree);data_offset=cursor;blob=bytearray(cursor)
for node in dirs:
 off=offsets[id(node)];struct.pack_into('<IIHHHH',blob,off,0,0,0,0,0,len(node))
 for n,(key,value) in enumerate(sorted(node.items())):
  target=(offsets[id(value)]|0x80000000) if isinstance(value,dict) else data_offset+16*value
  struct.pack_into('<II',blob,off+16+8*n,key,target)
def emit(data):return '\n'.join('.byte '+','.join(str(x) for x in data[i:i+24]) for i in range(0,len(data),24))
asm=['.section .rsrc,"dr"','.p2align 2',emit(blob)]
for i,data in enumerate(payloads):asm+=['.rva .Lpayload'+str(i),'.long '+str(len(data)),'.long 0','.long 0']
for i,data in enumerate(payloads):asm+=['.p2align 2','.Lpayload'+str(i)+':',emit(data)]
work=p/'build'/'resources';work.mkdir(parents=True,exist_ok=True);source=work/'icon-resource.s';source.write_text('\n'.join(asm)+'\n')
clang=shutil.which('clang')
if not clang: raise SystemExit('clang was not found; install LLVM or add clang to PATH')
subprocess.run([clang,'--target=x86_64-w64-windows-gnu','-c',str(source),'-o',str(p/'cmd'/'hame'/'hame_windows_amd64.syso')],check=True)
print('Embedded resource:',count,'icon sizes')
