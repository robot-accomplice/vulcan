"""AE6: row 19 warm-up rules in a model with a start-up excess (not a measurement of the base apps).
Per case (time constant tau in requests, noise sd): (% of runs with >2% excess in the timed window, % unsettled at the cap).
nb = neighbouring 50-request blocks (draft 10); 2v2 = last two blocks vs the two before, from block 4 (draft 11)."""
import numpy as np
rng=np.random.default_rng(1)
def run(tau,sd,rule,maxb,thr,N=3000):
    bad=uns=0
    for _ in range(N):
        n=np.arange(maxb*50)
        t=(1+1.0*np.exp(-n/tau))*np.exp(rng.normal(0,sd,len(n)))
        med=lambda a,b: np.median(t[a*50:b*50])
        stop=None
        for b in range(1,maxb):  # b = index of latest block (0-based), blocks 0..b observed
            if rule=='nb' and b>=1:
                ok=abs(med(b,b+1)/med(b-1,b)-1)<=thr
            elif rule=='3nb' and b>=2:
                ok=all(abs(med(i,i+1)/med(i-1,i)-1)<=thr for i in (b,b-1))
            elif rule=='2v2' and b>=3:
                ok=abs(med(b-1,b+1)/med(b-3,b-1)-1)<=thr
            else: ok=False
            if ok: stop=b+1;break
        if stop is None: uns+=1; stop=maxb
        s=stop*50
        k=np.arange(s,s+100)
        exc=np.mean(1.0*np.exp(-k/tau))
        if exc>0.02: bad+=1
    return round(100*bad/N),round(100*uns/N)
for rule,maxb,thr in [('nb',8,.05),('2v2',8,.05),('2v2',12,.05),('2v2',12,.10),('3nb',12,.05)]:
    print(rule,maxb,thr,[ (tau,sd,run(tau,sd,rule,maxb,thr,1500)) for tau,sd in [(60,.1),(60,.3),(100,.3),(200,.3),(60,.5)]])
print('---')
for rule,maxb,thr in [('2v2',16,.05),('2v2',16,.03),('2v2',12,.03)]:
    print(rule,maxb,thr,[ (tau,sd,run(tau,sd,rule,maxb,thr,1500)) for tau,sd in [(60,.1),(60,.3),(100,.3),(200,.3),(60,.5)]])
