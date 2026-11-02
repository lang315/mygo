import re,sys,statistics,collections
# usage: initagg.py file...   each file one run's stderr
runs=[]
for f in sys.argv[1:]:
    d={}; last=0
    for l in open(f,errors='replace'):
        m=re.match(r'init (\S+) @([\d.]+) ms, ([\d.]+) ms clock, (\d+) bytes, (\d+) allocs',l)
        if m:
            d[m.group(1)]=(float(m.group(3)),int(m.group(4)),int(m.group(5)),float(m.group(2))); last=max(last,float(m.group(2))+float(m.group(3)))
    d['__end__']=(last,0,0,last)
    runs.append(d)
pk=set().union(*[set(r) for r in runs])
rows=[]
for p in pk:
    v=[r[p] for r in runs if p in r]
    rows.append((statistics.median(x[0] for x in v),statistics.median(x[1] for x in v),statistics.median(x[2] for x in v),statistics.median(x[3] for x in v),p))
rows.sort(reverse=True)
tot=sum(r[0] for r in rows if r[4]!='__end__')
print('runs',len(runs),'sum of init clock %.2f ms; end of init (since runtime start) %.2f ms'%(tot,[r for r in rows if r[4]=='__end__'][0][0]))
for r in rows[:14]:
    if r[4]=='__end__': continue
    print('%7.3f ms %9d B %6d allocs  @%7.2f  %s'%(r[0],r[1],r[2],r[3],r[4]))
