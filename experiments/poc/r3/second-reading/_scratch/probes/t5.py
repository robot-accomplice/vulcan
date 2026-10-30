# T5: the same solver on the same body under three meanings of +.
# Go: wrap at 64 bits. Trap: a path that overflows stops, so the rest of the path holds no overflow.
# Go with the obligation (Gobra --overflow): the prover must also show that no overflow occurs.
import sys
sys.path.insert(0, 'py')
from z3 import *

def valid(f):
    s = Solver()
    s.add(Not(f))
    return s.check() == unsat

a, b = BitVecs('a b', 64)

def noov(x, y):
    return And(Implies(And(x > 0, y > 0), x + y > 0), Implies(And(x < 0, y < 0), x + y < 0))

cases = {
    'sum grows: b>=0 => a+b>=a': (b >= 0, a + b >= a, noov(a, b)),
    'midpoint: a<=b => a<=(a+b)/2<=b': (a <= b, And(a <= (a + b) / 2, (a + b) / 2 <= b), noov(a, b)),
    'refund: 0<=a,0<=b,b<=a => a-b>=0': (And(a >= 0, b >= 0, b <= a), a - b >= 0, BoolVal(True)),
    'total: 0<=a,0<=b => a+b>=0': (And(a >= 0, b >= 0), a + b >= 0, noov(a, b)),
}
print('%-40s %-8s %-8s %-10s' % ('property', 'Go wrap', 'Trap', 'Go+oblig'))
for k, (pre, post, ok) in cases.items():
    print('%-40s %-8s %-8s %-10s' % (k, valid(Implies(pre, post)), valid(Implies(And(pre, ok), post)), valid(Implies(pre, And(ok, post)))))

p, q = Ints('p q')
h = Array('h', IntSort(), IntSort())
print('pointers: *p=*p+1 leaves *q, no alias premise =', valid(Select(Store(h, p, Select(h, p) + 1), q) == Select(h, q)))
print('pointers: same, with premise p!=q =', valid(Implies(p != q, Select(Store(h, p, Select(h, p) + 1), q) == Select(h, q))))
