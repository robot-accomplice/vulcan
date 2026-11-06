"""AE6: row 19 warm-up rules in a model with a start-up excess (not a measurement of the base apps).
Per case (time constant tau in requests, noise sd): (% of runs with >2% excess in the timed window, % unsettled at the cap).
nb = neighbouring 50-request blocks (draft 10); 2v2 = last two blocks vs the two before, from block 4 (draft 11)."""
import numpy as np
rng=np.random.default_rng(1)
def run(tau,sd,rule,maxb,thr,N=3000,shape='exp',full=False):
    # shape 'exp': excess exp(-n/tau). shape 'pow': excess (1+n/50)^(-tau), tau is then the exponent p (AF5).
    ex=(lambda x: 1.0*np.exp(-x/tau)) if shape=='exp' else (lambda x: (1+x/50)**(-tau))
    bad=uns=0
    exs=[];falls=[];big=0
    for _ in range(N):
        n=np.arange(maxb*50+(100 if full else 0))  # full: 100 more requests, so the timed window exists at the cap
        t=(1+ex(n))*np.exp(rng.normal(0,sd,len(n)))
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
        exc=np.mean(ex(k))
        if exc>0.02: bad+=1
        if full:
            # AG4: residual fall = -ln(median timed 100 / median 100 before them); old measure = share with |ratio-1|>2%
            fall=-np.log(np.median(t[s:s+100])/np.median(t[s-100:s]))
            exs.append(exc);falls.append(fall);big+=abs(np.exp(-fall)-1)>0.02
    if full: return float(round(100*np.mean(exs),1)),float(round(100*np.mean(falls),1)),int(round(100*big/N))
    return round(100*bad/N),round(100*uns/N)
for rule,maxb,thr in [('nb',8,.05),('2v2',8,.05),('2v2',12,.05),('2v2',12,.10),('3nb',12,.05)]:
    print(rule,maxb,thr,[ (tau,sd,run(tau,sd,rule,maxb,thr,1500)) for tau,sd in [(60,.1),(60,.3),(100,.3),(200,.3),(60,.5)]])
print('---')
for rule,maxb,thr in [('2v2',16,.05),('2v2',16,.03),('2v2',12,.03)]:
    print(rule,maxb,thr,[ (tau,sd,run(tau,sd,rule,maxb,thr,1500)) for tau,sd in [(60,.1),(60,.3),(100,.3),(200,.3),(60,.5)]])
print('--- AF5: held-out power-law tails, excess (1+n/50)^(-p); added after the rule was fixed, not used to choose it')
for rule,maxb,thr in [('nb',8,.05),('2v2',12,.05)]:
    print(rule,maxb,thr,'pow',[ (p,sd,run(p,sd,rule,maxb,thr,1500,'pow')) for p,sd in [(0.5,.1),(0.5,.3),(0.5,.5),(1.0,.1),(1.0,.3),(1.0,.5)]])
print('--- AG4, AG5: mean excess left in the timed window (%), mean residual fall (%), old measure: share of runs with |ratio-1|>2% (%); no change of rule or cap; added after the rule was fixed, not used to choose it')
sds=[.1,.3,.5]
for rule,maxb,thr in [('nb',8,.05),('2v2',12,.05)]:
    print(rule,maxb,thr,'no drift',[ (sd,run(1e-9,sd,rule,maxb,thr,1500,'exp',True)) for sd in sds])
    for tau in (60,200):
        print(rule,maxb,thr,'exp tau',tau,[ (sd,run(tau,sd,rule,maxb,thr,1500,'exp',True)) for sd in sds])
    for p in (0.5,1.0,1.5,2.0):
        print(rule,maxb,thr,'pow p',p,[ (sd,run(p,sd,rule,maxb,thr,1500,'pow',True)) for sd in sds])
print('--- AG5: share of runs (%) with >2% excess left, power-law p = 1.5 and 2.0')
for rule,maxb,thr in [('nb',8,.05),('2v2',12,.05)]:
    print(rule,maxb,thr,'pow',[ (p,sd,run(p,sd,rule,maxb,thr,1500,'pow')) for p in (1.5,2.0) for sd in sds])
